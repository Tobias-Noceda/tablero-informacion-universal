package executer

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
)

func cardAt(t *testing.T, address string, query map[string]string) *models.PostIts {
	t.Helper()
	resource, err := url.Parse(address)
	if err != nil {
		t.Fatal(err)
	}
	return &models.PostIts{
		Resource: resource,
		Request:  models.Request{Method: http.MethodGet},
		Response: "json",
		Query:    query,
	}
}

func answering(t *testing.T, status int, body string) string {
	t.Helper()
	allowLoopback(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestExecute_NamesTheSourcesStatus(t *testing.T) {
	_, err := New().Execute(cardAt(t, answering(t, http.StatusNotFound, ""), nil))

	var status *infrastructure.SourceStatusError
	if !errors.As(err, &status) || status.Status != http.StatusNotFound {
		t.Fatalf("err = %v, want the source's 404", err)
	}
	// What an agent reading the error through the MCP server sees.
	if err.Error() != "Resource returned 404" {
		t.Errorf("message = %q", err.Error())
	}
}

func TestExecute_ASourceNobodyListensOnIsUnreachable(t *testing.T) {
	allowLoopback(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := "http://" + listener.Addr().String()
	listener.Close()

	_, err = New().Execute(cardAt(t, address, nil))

	if !errors.Is(err, infrastructure.ErrSourceUnreachable) {
		t.Errorf("err = %v, want unreachable", err)
	}
}

func TestExecute_AnAnswerThatIsNotJSONIsUnreadable(t *testing.T) {
	_, err := New().Execute(cardAt(t, answering(t, http.StatusOK, "<html>not json</html>"), map[string]string{"a": ".a"}))

	if !errors.Is(err, infrastructure.ErrSourceUnreadable) {
		t.Errorf("err = %v, want unreadable", err)
	}
}

func TestExecute_QueryFailures(t *testing.T) {
	for name, query := range map[string]string{
		"no result":    "empty",
		"bad syntax":   ".a[",
		"runtime fail": ".a | error(\"nope\")",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := New().Execute(cardAt(t, answering(t, http.StatusOK, `{"a":1}`), map[string]string{"out": query}))

			if !errors.Is(err, infrastructure.ErrQueryFailed) {
				t.Errorf("err = %v, want a query failure", err)
			}
		})
	}
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "i/o timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

func TestTransportFailure_TellsATimeoutFromTheRest(t *testing.T) {
	for name, c := range map[string]struct {
		err  error
		want error
	}{
		"deadline":        {fmt.Errorf("get: %w", context.DeadlineExceeded), infrastructure.ErrSourceTimeout},
		"network timeout": {&url.Error{Op: "Get", URL: "http://x", Err: timeoutError{}}, infrastructure.ErrSourceTimeout},
		"refused":         {errors.New("connection refused"), infrastructure.ErrSourceUnreachable},
		"blocked address": {errors.New("Failed to connect"), infrastructure.ErrSourceUnreachable},
	} {
		t.Run(name, func(t *testing.T) {
			if got := transportFailure(c.err); !errors.Is(got, c.want) {
				t.Errorf("transportFailure(%v) = %v, want %v", c.err, got, c.want)
			}
		})
	}
}
