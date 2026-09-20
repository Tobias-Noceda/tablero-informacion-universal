package postits

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/ports/crypto"
	"github.com/Secreto31126/tesis/common/ports/executer"
	"github.com/Secreto31126/tesis/common/ports/safehttp"
	secretsrv "github.com/Secreto31126/tesis/common/services/secrets"
	"github.com/google/uuid"
)

const nasaKey = "nasa_live_SUPERSECRET"

func allowLoopback(t *testing.T) {
	t.Helper()
	original := safehttp.IsSafeIP
	safehttp.IsSafeIP = func(net.IP) bool { return true }
	t.Cleanup(func() { safehttp.IsSafeIP = original })
}

// realSecrets wires a genuine sealer, keyring and secrets service over an
// in-memory store, so the value really is encrypted at rest.
func realSecrets(t *testing.T) *secretsrv.SecretsService {
	t.Helper()
	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x6A}, 32))
	sealer, err := crypto.NewFromKeys("1:" + key)
	if err != nil {
		t.Fatalf("sealer: %v", err)
	}

	var rows []models.Secret
	store := &mocks.MockSecretStore{
		UpsertSecretFn: func(s *models.Secret) error {
			rows = append(rows, *s)
			return nil
		},
		FindSecretsFn: func(scope models.SecretScope, names []string) ([]models.Secret, error) {
			var out []models.Secret
			for _, row := range rows {
				if row.Scope == scope && slices.Contains(names, row.Name) {
					out = append(out, row)
				}
			}
			return out, nil
		},
	}

	return secretsrv.New(store, &mocks.MockScopePolicy{}, crypto.NewKeyring(sealer, &mocks.MemoryKeyStore{}),
		&mocks.MockTokenClient{}, &mocks.MockLocker{}, &mocks.MockHandshakeStore{})
}

// nasaStub stands in for api.nasa.gov and records the api_key it was sent.
func nasaStub(t *testing.T) (*httptest.Server, *string) {
	t.Helper()
	var gotKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.URL.Query().Get("api_key")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"title":"Pillars of Creation","url":"https://apod.nasa.gov/pillars.jpg","explanation":"Gas and dust.","date":"2026-09-19"}`)
	}))
	t.Cleanup(server.Close)
	return server, &gotKey
}

// createNasaPostIt goes through CreatePostIt like the API does, then points
// the stored post-it at the stub the way a persisted resource would be read.
func createNasaPostIt(t *testing.T, svc *PostItsService, board uuid.UUID, resource string, params map[string]string) *models.PostIts {
	t.Helper()
	created, err := svc.CreatePostIt(boardOwner, &models.PostIts{Board: board, WellKnown: "nasa_apod", Params: params})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	created.Id = uuid.New()
	created.Resource, _ = url.Parse(resource)
	return created
}

func TestWellKnowns_SystemSecretsAreKnownToTheCode(t *testing.T) {
	for key, wk := range configuredPostIts {
		for placeholder, name := range wk.systemSecrets {
			if !slices.Contains(models.KnownSystemSecrets, name) {
				t.Errorf("%s: %s references %q, which is not in KnownSystemSecrets", key, placeholder, name)
			}
			if _, isRef := strings.CutPrefix(placeholder, "$"); !isRef || models.ValidSecretName(placeholder[1:]) {
				t.Errorf("%s: placeholder %q must be a lowercase $name so it is never mistaken for a user secret", key, placeholder)
			}
		}
	}
}

func TestNasaApod_InjectsTheSystemKeyAtExecution(t *testing.T) {
	allowLoopback(t)
	provider, gotKey := nasaStub(t)
	secrets := realSecrets(t)
	if err := secrets.Put(models.SystemScope, models.Principal{}, string(models.SystemNasaApiKey), models.SecretApiKey, nasaKey); err != nil {
		t.Fatalf("put system secret: %v", err)
	}

	svc := New(boardOf(boardOwner), &mocks.MockCache{}, executer.New(), secrets)
	postit := createNasaPostIt(t, svc, uuid.New(), provider.URL, nil)

	data, err := svc.ExecutePostIt(postit)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if *gotKey != nasaKey {
		t.Errorf("provider saw api_key=%q, want the system secret", *gotKey)
	}

	result, ok := data.(map[string]any)
	if !ok || result["title"] != "Pillars of Creation" {
		t.Errorf("result = %v", data)
	}
}

// What is persisted and what the client reads back must carry neither the
// system secret's name nor its value.
func TestNasaApod_StoredPostItNeverNamesTheSystemSecret(t *testing.T) {
	allowLoopback(t)
	provider, _ := nasaStub(t)
	secrets := realSecrets(t)
	_ = secrets.Put(models.SystemScope, models.Principal{}, string(models.SystemNasaApiKey), models.SecretApiKey, nasaKey)

	svc := New(boardOf(boardOwner), &mocks.MockCache{}, executer.New(), secrets)
	postit := createNasaPostIt(t, svc, uuid.New(), provider.URL, nil)

	if _, err := svc.ExecutePostIt(postit); err != nil {
		t.Fatalf("execute: %v", err)
	}

	encoded, _ := json.Marshal(postit)
	for _, leak := range []string{nasaKey, string(models.SystemNasaApiKey)} {
		if bytes.Contains(encoded, []byte(leak)) {
			t.Errorf("the post-it the client sees contains %q: %s", leak, encoded)
		}
	}
	if postit.Request.Queries["api_key"] != "$api_key" {
		t.Errorf("stored query = %q, want the placeholder", postit.Request.Queries["api_key"])
	}
}

// A user editing params cannot substitute their own value for the platform's.
func TestNasaApod_UserParamsCannotShadowTheSystemSecret(t *testing.T) {
	allowLoopback(t)
	provider, gotKey := nasaStub(t)
	secrets := realSecrets(t)
	_ = secrets.Put(models.SystemScope, models.Principal{}, string(models.SystemNasaApiKey), models.SecretApiKey, nasaKey)

	svc := New(boardOf(boardOwner), &mocks.MockCache{}, executer.New(), secrets)
	postit := createNasaPostIt(t, svc, uuid.New(), provider.URL, nil)
	postit.Params = map[string]string{"$api_key": "attacker-chosen"}

	if _, err := svc.ExecutePostIt(postit); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if *gotKey != nasaKey {
		t.Errorf("provider saw api_key=%q, want the system secret to win", *gotKey)
	}
}

// Only a well-known definition can reach the system scope. A hand-made
// post-it naming the secret gets the reference sent verbatim, like any secret
// it does not own.
func TestCustomPostIt_CannotReferenceASystemSecret(t *testing.T) {
	allowLoopback(t)
	provider, gotKey := nasaStub(t)
	secrets := realSecrets(t)
	_ = secrets.Put(models.SystemScope, models.Principal{}, string(models.SystemNasaApiKey), models.SecretApiKey, nasaKey)

	resource, _ := url.Parse(provider.URL)
	postit := &models.PostIts{
		Id:       uuid.New(),
		Board:    uuid.New(),
		Resource: resource,
		Request: models.Request{
			Method:  http.MethodGet,
			Queries: map[string]string{"api_key": "$NASA_API_KEY"},
		},
		Response: "json",
		Query:    map[string]string{"title": ".title"},
	}

	svc := New(boardOf(boardOwner), &mocks.MockCache{}, executer.New(), secrets)
	if _, err := svc.ExecutePostIt(postit); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if *gotKey != "$NASA_API_KEY" {
		t.Errorf("provider saw api_key=%q, want the unresolved reference", *gotKey)
	}
}

// The provider is never called when the platform has not provisioned the
// secret, and the error handed back does not say which secret is missing.
func TestNasaApod_MissingSystemSecretFailsWithoutNamingIt(t *testing.T) {
	allowLoopback(t)
	provider, gotKey := nasaStub(t)
	called := false
	provider.Config.Handler = http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })

	svc := New(boardOf(boardOwner), &mocks.MockCache{}, executer.New(), realSecrets(t))
	postit := createNasaPostIt(t, svc, uuid.New(), provider.URL, nil)

	_, err := svc.ExecutePostIt(postit)
	if !errors.Is(err, ErrSystemSecretMissing) {
		t.Fatalf("err = %v, want ErrSystemSecretMissing", err)
	}
	if strings.Contains(err.Error(), string(models.SystemNasaApiKey)) {
		t.Errorf("the error names the missing secret: %v", err)
	}
	if called || *gotKey != "" {
		t.Error("the provider was called without the credential")
	}
}
