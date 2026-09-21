package users

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	srv "github.com/Secreto31126/tesis/common/services/users"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var (
	ana = models.User{Id: uuid.New(), Email: "ana@example.com", Name: "Ana", EmailVerified: true,
		Identities: []models.Identity{{Provider: models.IdentityPassword, Hash: "plain:secret"}}}
	bob = models.User{Id: uuid.New(), Email: "bob@example.com", Name: "Bob", Admin: true}
)

func setup() *gin.Engine {
	store := &mocks.MemoryUserStore{Users: []models.User{ana, bob}}
	verifier := mocks.StaticVerifier{"ana": infrastructure.Claims{Subject: ana.Id.String()}}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewController(srv.New(store), verifier).RegisterRoutes(r)
	return r
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

func TestGetUser_OwnProfileIsFull(t *testing.T) {
	r := setup()

	w := do(r, http.MethodGet, "/users/"+ana.Id.String(), "", "ana")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{`"email":"ana@example.com"`, `"email_verified":true`, `"identities":["password"]`, `"admin":false`} {
		if !strings.Contains(body, want) {
			t.Errorf("body %s lacks %s", body, want)
		}
	}
	if strings.Contains(body, "plain:") {
		t.Error("hash leaked")
	}
}

func TestGetUser_SomeoneElseIsASummary(t *testing.T) {
	r := setup()

	w := do(r, http.MethodGet, "/users/"+bob.Id.String(), "", "ana")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"name":"Bob"`) || strings.Contains(body, "admin") || strings.Contains(body, "identities") || strings.Contains(body, "email_verified") {
		t.Errorf("summary leaks private fields: %s", body)
	}
}

func TestGetUser_UnknownAndAnonymous(t *testing.T) {
	r := setup()

	if w := do(r, http.MethodGet, "/users/"+uuid.NewString(), "", "ana"); w.Code != http.StatusNotFound {
		t.Errorf("unknown: %d", w.Code)
	}
	if w := do(r, http.MethodGet, "/users/not-a-uuid", "", "ana"); w.Code != http.StatusBadRequest {
		t.Errorf("bad id: %d", w.Code)
	}
	if w := do(r, http.MethodGet, "/users/"+ana.Id.String(), "", ""); w.Code != http.StatusUnauthorized {
		t.Errorf("anonymous: %d", w.Code)
	}
}

func TestPatchUser_RenamesOnlyOneself(t *testing.T) {
	r := setup()

	w := do(r, http.MethodPatch, "/users/"+ana.Id.String(), `{"name":"Ana María"}`, "ana")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"name":"Ana María"`) {
		t.Errorf("rename: %d %s", w.Code, w.Body.String())
	}
	if w := do(r, http.MethodPatch, "/users/"+bob.Id.String(), `{"name":"Eve"}`, "ana"); w.Code != http.StatusNotFound {
		t.Errorf("someone else: %d", w.Code)
	}
	if w := do(r, http.MethodPatch, "/users/"+ana.Id.String(), `{"name":"  "}`, "ana"); w.Code != http.StatusBadRequest {
		t.Errorf("blank: %d", w.Code)
	}
	if w := do(r, http.MethodPatch, "/users/"+ana.Id.String(), `{}`, "ana"); w.Code != http.StatusBadRequest {
		t.Errorf("missing: %d", w.Code)
	}
}
