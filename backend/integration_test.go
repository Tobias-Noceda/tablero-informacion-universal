//go:build integration

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/ports/crypto"
	"github.com/Secreto31126/tesis/common/ports/mongo"
	"github.com/Secreto31126/tesis/common/ports/redis"
	"github.com/Secreto31126/tesis/common/ports/safehttp"
	"github.com/gin-gonic/gin"
)

const (
	nasaKey   = "nasa_live_ONLY_THE_PROVIDER_MAY_SEE_THIS"
	testOwner = "it-owner"
)

// stack wires the real application over the Mongo and Redis named by the
// environment (see run_backend_integration.*) and records every response so
// the test can prove none of them carried a secret.
type stack struct {
	t         *testing.T
	app       *app
	db        *mongo.MongoDB
	responses []*httptest.ResponseRecorder
}

func newStack(t *testing.T) *stack {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := mongo.New()
	if err != nil {
		t.Fatalf("mongo: %v", err)
	}
	if err := db.EnsureIndexes(); err != nil {
		t.Skipf("mongo not reachable: %v", err)
	}

	cache, err := redis.New()
	if err != nil {
		t.Fatalf("redis: %v", err)
	}

	kek, err := crypto.NewFromKeys("1:" + base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x11}, 32)))
	if err != nil {
		t.Fatalf("kek: %v", err)
	}

	t.Cleanup(func() {
		_ = cache.Close()
		_ = db.Close()
	})

	return &stack{t: t, app: newApp(db, cache, kek), db: db}
}

func (s *stack) do(method, path string, body any) *httptest.ResponseRecorder {
	s.t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, _ := json.Marshal(body)
		reader = bytes.NewReader(encoded)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.app.router.ServeHTTP(w, req)
	s.responses = append(s.responses, w)
	return w
}

func (s *stack) mustJSON(w *httptest.ResponseRecorder, want int, out any) {
	s.t.Helper()
	if w.Code != want {
		s.t.Fatalf("status = %d, want %d (body: %s)", w.Code, want, w.Body.String())
	}
	if out != nil {
		if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
			s.t.Fatalf("decode %s: %v", w.Body.String(), err)
		}
	}
}

func (s *stack) assertNoResponseContains(needle string) {
	s.t.Helper()
	for i, w := range s.responses {
		if strings.Contains(w.Body.String(), needle) {
			s.t.Errorf("response #%d carried %q: %s", i, needle, w.Body.String())
		}
	}
}

func TestEndToEnd_SystemSecretReachesTheProviderAndNoClient(t *testing.T) {
	original := safehttp.IsSafeIP
	safehttp.IsSafeIP = func(net.IP) bool { return true }
	t.Cleanup(func() { safehttp.IsSafeIP = original })

	var seenKey string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenKey = r.URL.Query().Get("api_key")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"title":"Pillars of Creation","url":"https://apod.nasa.gov/p.jpg","explanation":"Gas.","date":"2026-09-19"}`)
	}))
	defer provider.Close()

	s := newStack(t)

	// The platform provisions its key; the listing shows it is there.
	s.mustJSON(s.do(http.MethodPut, "/api/v1/system/secrets",
		map[string]string{"name": string(models.SystemNasaApiKey), "kind": "api_key", "value": nasaKey}), http.StatusNoContent, nil)

	var statuses []models.SystemSecretStatus
	s.mustJSON(s.do(http.MethodGet, "/api/v1/system/secrets", nil), http.StatusOK, &statuses)
	configured := false
	for _, st := range statuses {
		configured = configured || (st.Name == string(models.SystemNasaApiKey) && st.Configured)
	}
	if !configured {
		t.Fatalf("NASA_API_KEY not reported configured: %+v", statuses)
	}

	// A user builds a board and drops the card on it.
	var board models.Board
	s.mustJSON(s.do(http.MethodPost, "/api/v1/boards", map[string]string{"name": "it", "owner": testOwner}), http.StatusCreated, &board)

	var postit models.PostIts
	s.mustJSON(s.do(http.MethodPost, "/api/v1/post-its",
		map[string]any{"board": board.Id, "well_known": "nasa_apod"}), http.StatusCreated, &postit)

	if postit.Request.Queries["api_key"] != "$api_key" {
		t.Fatalf("stored query = %q, want the placeholder", postit.Request.Queries["api_key"])
	}

	// Point the persisted card at the stub, the way it would be read back from
	// the database, without touching the request template.
	stub, _ := url.Parse(provider.URL)
	if err := s.db.UpdatePostIt(postit.Id, map[string]any{"resource": stub}); err != nil {
		t.Fatalf("retarget: %v", err)
	}

	var settings models.PostIts
	settingsResponse := s.do(http.MethodGet, "/api/v1/post-its/"+postit.Id.String()+"/settings", nil)
	s.mustJSON(settingsResponse, http.StatusOK, &settings)

	var result map[string]any
	executionResponse := s.do(http.MethodGet, "/api/v1/post-its/"+postit.Id.String(), nil)
	s.mustJSON(executionResponse, http.StatusOK, &result)

	if seenKey != nasaKey {
		t.Errorf("provider saw api_key=%q, want the system secret", seenKey)
	}
	if result["title"] != "Pillars of Creation" {
		t.Errorf("result = %v", result)
	}

	// Executing again is served from Redis; the provider is not called twice.
	seenKey = ""
	s.mustJSON(s.do(http.MethodGet, "/api/v1/post-its/"+postit.Id.String(), nil), http.StatusOK, &result)
	if seenKey != "" {
		t.Error("the second execution hit the provider instead of the cache")
	}

	// Deleting the board purges the scope, and the platform's key stays.
	s.mustJSON(s.do(http.MethodDelete, "/api/v1/boards/"+board.Id.String(), nil), http.StatusNoContent, nil)
	if _, err := s.db.FindActiveKey(models.BoardScope(board.Id)); err == nil {
		t.Error("the board's data key survived the delete")
	}
	if _, err := s.db.FindActiveKey(models.SystemScope); err != nil {
		t.Errorf("the system data key is gone: %v", err)
	}

	// The value never leaves the server. The name only appears in the operator
	// listing under /system; what a board user gets back never mentions it.
	s.assertNoResponseContains(nasaKey)
	for _, w := range []*httptest.ResponseRecorder{settingsResponse, executionResponse} {
		if strings.Contains(w.Body.String(), string(models.SystemNasaApiKey)) {
			t.Errorf("a user-facing response names the system secret: %s", w.Body.String())
		}
	}

	s.mustJSON(s.do(http.MethodDelete, "/api/v1/system/secrets/"+string(models.SystemNasaApiKey), nil), http.StatusNoContent, nil)
}

// Without the platform key the card is unavailable, and the failure does not
// say which credential is missing.
func TestEndToEnd_MissingSystemSecretIsUnavailable(t *testing.T) {
	s := newStack(t)
	s.do(http.MethodDelete, "/api/v1/system/secrets/"+string(models.SystemNasaApiKey), nil)

	var board models.Board
	s.mustJSON(s.do(http.MethodPost, "/api/v1/boards", map[string]string{"name": "it", "owner": testOwner}), http.StatusCreated, &board)
	defer s.do(http.MethodDelete, "/api/v1/boards/"+board.Id.String(), nil)

	var postit models.PostIts
	s.mustJSON(s.do(http.MethodPost, "/api/v1/post-its",
		map[string]any{"board": board.Id, "well_known": "nasa_apod"}), http.StatusCreated, &postit)

	w := s.do(http.MethodGet, "/api/v1/post-its/"+postit.Id.String(), nil)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (body: %s)", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "NASA") {
		t.Errorf("the error names the secret: %s", w.Body.String())
	}
}

// A board's owner can list what exists but never read a value back, and a
// stranger cannot even learn the board exists.
func TestEndToEnd_BoardSecretsAreWriteOnly(t *testing.T) {
	s := newStack(t)

	var board models.Board
	s.mustJSON(s.do(http.MethodPost, "/api/v1/boards", map[string]string{"name": "it", "owner": testOwner}), http.StatusCreated, &board)
	defer s.do(http.MethodDelete, "/api/v1/boards/"+board.Id.String(), nil)
	base := "/api/v1/boards/" + board.Id.String()

	s.mustJSON(s.do(http.MethodPut, base+"/secrets",
		map[string]string{"cognito_id": testOwner, "name": "MY_KEY", "kind": "bearer", "value": "board-secret-value"}), http.StatusNoContent, nil)

	var metas []models.SecretMeta
	s.mustJSON(s.do(http.MethodGet, base+"/secrets?cognito_id="+testOwner, nil), http.StatusOK, &metas)
	if len(metas) != 1 || metas[0].Name != "MY_KEY" || metas[0].Scope != models.BoardScope(board.Id) {
		t.Errorf("metas = %+v", metas)
	}

	if w := s.do(http.MethodGet, base+"/secrets?cognito_id=stranger", nil); w.Code != http.StatusNotFound {
		t.Errorf("stranger got %d, want 404", w.Code)
	}

	s.assertNoResponseContains("board-secret-value")
}
