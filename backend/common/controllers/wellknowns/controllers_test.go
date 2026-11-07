package wellknowns

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Secreto31126/tesis/common/controllers/middleware"
	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	srv "github.com/Secreto31126/tesis/common/services/wellknowns"
	"github.com/gin-gonic/gin"
)

// tokenVerifier lets a test call as a personal access token: "pat:<id>".
type tokenVerifier struct{ mocks.SubjectVerifier }

func (v tokenVerifier) Verify(token string) (infrastructure.Claims, error) {
	if id, ok := strings.CutPrefix(token, "pat:"); ok {
		return infrastructure.Claims{Subject: id, AccessToken: true}, nil
	}
	return v.SubjectVerifier.Verify(token)
}

func setup(t *testing.T) http.Handler {
	t.Helper()
	gin.SetMode(gin.TestMode)

	service := srv.New(mocks.NewMemoryWellKnownStore())
	if err := service.Seed(); err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	api := r.Group("/api/v1", middleware.RequireAuth(tokenVerifier{}), middleware.RestrictAccessTokens())
	NewController(service).RegisterRoutes(api)
	return r
}

func do(r http.Handler, token, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/api/v1"+path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func dogFacts(t *testing.T) string {
	t.Helper()
	for _, wk := range srv.Builtin() {
		if wk.Key == "dog_facts" {
			body, _ := json.Marshal(wk)
			return string(body)
		}
	}
	t.Fatal("no dog_facts")
	return ""
}

func TestReads_AnySignedInUser(t *testing.T) {
	r := setup(t)

	if w := do(r, "", http.MethodGet, "/well-knowns", ""); w.Code != http.StatusUnauthorized {
		t.Errorf("anonymous list = %d", w.Code)
	}

	for _, token := range []string{"ana", "pat:ana"} {
		w := do(r, token, http.MethodGet, "/well-knowns", "")
		if w.Code != http.StatusOK {
			t.Fatalf("%s list = %d %s", token, w.Code, w.Body)
		}
		var wks []models.WellKnown
		if err := json.Unmarshal(w.Body.Bytes(), &wks); err != nil || len(wks) != len(srv.Builtin()) {
			t.Errorf("%s listed %d (%v)", token, len(wks), err)
		}
		if wks[0].Key != "static_card" {
			t.Errorf("first = %s, want the picker order", wks[0].Key)
		}

		if w := do(r, token, http.MethodGet, "/well-knowns/nasa_apod", ""); w.Code != http.StatusOK || strings.Contains(w.Body.String(), "NASA_API_KEY") {
			t.Errorf("%s get = %d %s, want it without the system secret", token, w.Code, w.Body)
		}
	}

	if w := do(r, "ana", http.MethodGet, "/well-knowns/nope", ""); w.Code != http.StatusNotFound {
		t.Errorf("missing = %d", w.Code)
	}
}

func TestWrites_AdminOnly(t *testing.T) {
	r := setup(t)
	body := dogFacts(t)

	for _, token := range []string{"ana", "pat:ana"} {
		for _, c := range []struct{ method, path, body string }{
			{http.MethodPut, "/well-knowns/cats", body},
			{http.MethodDelete, "/well-knowns/dog_facts", ""},
			{http.MethodPost, "/well-knowns/dog_facts/restore", ""},
		} {
			if w := do(r, token, c.method, c.path, c.body); w.Code != http.StatusForbidden {
				t.Errorf("%s %s %s = %d, want 403", token, c.method, c.path, w.Code)
			}
		}
	}

	if w := do(r, "admin:root", http.MethodPut, "/well-knowns/cats", body); w.Code != http.StatusOK {
		t.Fatalf("admin put = %d %s", w.Code, w.Body)
	}
	if w := do(r, "ana", http.MethodGet, "/well-knowns/cats", ""); w.Code != http.StatusOK {
		t.Errorf("get new = %d", w.Code)
	}

	if w := do(r, "admin:root", http.MethodPut, "/well-knowns/cats", `{"key":"cats"}`); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "invalid_well_known") {
		t.Errorf("invalid put = %d %s", w.Code, w.Body)
	}

	if w := do(r, "admin:root", http.MethodDelete, "/well-knowns/cats", ""); w.Code != http.StatusNoContent {
		t.Errorf("delete = %d", w.Code)
	}
	if w := do(r, "ana", http.MethodGet, "/well-knowns/cats", ""); w.Code != http.StatusNotFound {
		t.Errorf("get deleted = %d", w.Code)
	}
	if w := do(r, "admin:root", http.MethodPost, "/well-knowns/cats/restore", ""); w.Code != http.StatusOK {
		t.Errorf("restore = %d %s", w.Code, w.Body)
	}
	if w := do(r, "admin:root", http.MethodPost, "/well-knowns/cats/restore", ""); w.Code != http.StatusConflict {
		t.Errorf("restore live = %d", w.Code)
	}
}
