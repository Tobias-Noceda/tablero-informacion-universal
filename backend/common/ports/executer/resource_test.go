package executer

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
)

// echoPath answers the path and query it was asked for.
func echoPath(t *testing.T) *httptest.Server {
	t.Helper()
	allowLoopback(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"path":%q,"query":%q}`, r.URL.EscapedPath(), r.URL.RawQuery)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func run(t *testing.T, resource string, params map[string]string) (map[string]any, error) {
	t.Helper()
	parsed, err := url.Parse(resource)
	if err != nil {
		t.Fatal(err)
	}
	data, err := New().Execute(&models.PostIts{
		Resource: parsed,
		Request:  models.Request{Method: http.MethodGet},
		Response: "json",
		Params:   params,
		Query:    map[string]string{"path": ".path", "query": ".query"},
	})
	if err != nil {
		return nil, err
	}
	return data.(map[string]any), nil
}

func TestExecute_FillsTheAddressPath(t *testing.T) {
	srv := echoPath(t)

	got, err := run(t, srv.URL+"/repos/$repo/releases/latest", map[string]string{"$repo": "sveltejs/svelte"})
	if err != nil {
		t.Fatal(err)
	}
	if got["path"] != "/repos/sveltejs/svelte/releases/latest" {
		t.Errorf("path = %v", got["path"])
	}
}

func TestExecute_EscapesWhatAPathValueCarries(t *testing.T) {
	srv := echoPath(t)

	got, err := run(t, srv.URL+"/holidays/$country", map[string]string{"$country": "AR?admin=1#x y"})
	if err != nil {
		t.Fatal(err)
	}
	if got["query"] != "" {
		t.Errorf("a path value reached the query string: %v", got["query"])
	}
	if got["path"] != "/holidays/AR%3Fadmin=1%23x%20y" {
		t.Errorf("path = %v", got["path"])
	}
}

func TestExecute_RefusesAPathValueThatClimbsOut(t *testing.T) {
	srv := echoPath(t)

	for _, value := range []string{"..", "../admin", "a/../../b", "."} {
		if _, err := run(t, srv.URL+"/api/$id/info", map[string]string{"$id": value}); !errors.Is(err, infrastructure.ErrSourceUnreachable) {
			t.Errorf("%q: got %v, want the address refused", value, err)
		}
	}
}

func TestExecute_FillsTheAddressHost(t *testing.T) {
	srv := echoPath(t)
	host := strings.TrimPrefix(srv.URL, "http://")

	got, err := run(t, "http://$domain/api/v2/status.json", map[string]string{"$domain": host})
	if err != nil {
		t.Fatal(err)
	}
	if got["path"] != "/api/v2/status.json" {
		t.Errorf("path = %v", got["path"])
	}

	for _, value := range []string{"evil.example/x", "user@evil.example", "a b", ""} {
		if _, err := run(t, "http://$domain/x", map[string]string{"$domain": value}); !errors.Is(err, infrastructure.ErrSourceUnreachable) {
			t.Errorf("host %q: got %v, want the address refused", value, err)
		}
	}
}

func TestExecute_LeavesAnUnfilledPlaceholder(t *testing.T) {
	srv := echoPath(t)

	got, err := run(t, srv.URL+"/items/$id", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got["path"] != "/items/$id" {
		t.Errorf("path = %v", got["path"])
	}
}

// A host value is still checked where the request goes: a card type cannot be
// pointed at the platform's own network through a field.
func TestExecute_AHostValueStillPassesTheSSRFGuard(t *testing.T) {
	parsed, _ := url.Parse("http://$domain/admin")
	_, err := New().Execute(&models.PostIts{
		Resource: parsed,
		Request:  models.Request{Method: http.MethodGet},
		Response: "json",
		Params:   map[string]string{"$domain": "127.0.0.1"},
		Query:    map[string]string{"x": ".x"},
	})
	if !errors.Is(err, infrastructure.ErrSourceUnreachable) {
		t.Errorf("got %v, want the internal address refused", err)
	}
}
