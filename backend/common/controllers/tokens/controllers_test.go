package tokens

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/controllers/middleware"
	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/mocks"
	srv "github.com/Secreto31126/tesis/common/services/tokens"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var (
	ana = uuid.NewString()
	bob = uuid.NewString()
)

// setup mounts the controller the way app.go does: behind RequireAuth with
// the token verifier, and behind RestrictAccessTokens.
func setup() (*gin.Engine, *mocks.MemoryAccessTokenStore) {
	store := &mocks.MemoryAccessTokenStore{}
	service := srv.New(store, &mocks.CountingLimiter{}, time.Now)
	verifier := service.Verifier(mocks.StaticVerifier{
		"ana": infrastructure.Claims{Subject: ana},
		"bob": infrastructure.Claims{Subject: bob},
	})
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewController(service).RegisterRoutes(r.Group("/api/v1", middleware.RequireAuth(verifier), middleware.RestrictAccessTokens()))
	return r, store
}

func do(r http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestTokens_CreateListRevoke(t *testing.T) {
	r, _ := setup()
	base := "/api/v1/users/" + ana + "/tokens"

	w := do(r, http.MethodPost, base, `{"name":"laptop"}`, "ana")
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var created srv.Created
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	if !strings.HasPrefix(created.Token, srv.PREFIX) || strings.Contains(w.Body.String(), `"hash"`) || strings.Contains(w.Body.String(), `"user"`) {
		t.Errorf("create body = %s", w.Body.String())
	}

	w = do(r, http.MethodGet, base, "", "ana")
	if w.Code != http.StatusOK || strings.Contains(w.Body.String(), created.Token) || !strings.Contains(w.Body.String(), created.Prefix) {
		t.Errorf("list: %d %s", w.Code, w.Body.String())
	}

	// The token itself may not manage tokens.
	if w := do(r, http.MethodGet, base, "", created.Token); w.Code != http.StatusForbidden {
		t.Errorf("list with the token: %d", w.Code)
	}
	if w := do(r, http.MethodPost, base, `{"name":"more"}`, created.Token); w.Code != http.StatusForbidden {
		t.Errorf("create with the token: %d", w.Code)
	}

	if w := do(r, http.MethodDelete, base+"/"+created.Id.String(), "", "ana"); w.Code != http.StatusNoContent {
		t.Errorf("revoke: %d", w.Code)
	}
	if w := do(r, http.MethodGet, base, "", created.Token); w.Code != http.StatusUnauthorized {
		t.Errorf("revoked token: %d", w.Code)
	}
}

func TestTokens_OnlyOneself(t *testing.T) {
	r, _ := setup()
	created := do(r, http.MethodPost, "/api/v1/users/"+ana+"/tokens", `{"name":"laptop"}`, "ana")
	var token srv.Created
	_ = json.Unmarshal(created.Body.Bytes(), &token)

	if w := do(r, http.MethodGet, "/api/v1/users/"+ana+"/tokens", "", "bob"); w.Code != http.StatusNotFound {
		t.Errorf("someone else's list: %d", w.Code)
	}
	if w := do(r, http.MethodDelete, "/api/v1/users/"+bob+"/tokens/"+token.Id.String(), "", "bob"); w.Code != http.StatusNotFound {
		t.Errorf("someone else's token: %d", w.Code)
	}
	if w := do(r, http.MethodGet, "/api/v1/users/"+ana+"/tokens", "", ""); w.Code != http.StatusUnauthorized {
		t.Errorf("anonymous: %d", w.Code)
	}
}

func TestTokens_Validation(t *testing.T) {
	r, _ := setup()
	base := "/api/v1/users/" + ana + "/tokens"

	cases := map[string]string{
		`{}`:                       "invalid_name",
		`{"name":"  "}`:            "invalid_name",
		`{"name":"x","days":365}`:  "invalid_expiry",
		`{"name":"x","days":"15"}`: "invalid_request",
	}
	for body, want := range cases {
		w := do(r, http.MethodPost, base, body, "ana")
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), want) {
			t.Errorf("%s: %d %s", body, w.Code, w.Body.String())
		}
	}
}
