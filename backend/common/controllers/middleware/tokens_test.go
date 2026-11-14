package middleware

import (
	"errors"
	"fmt"
	"github.com/Secreto31126/tesis/common/models"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/gin-gonic/gin"
)

func restricted() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	verifier := mocks.StaticVerifier{
		"session": {Subject: "user-1"},
		"pat":     {Subject: "user-1", AccessToken: true},
	}
	api := r.Group("/api/v1", RequireAuth(verifier), RestrictAccessTokens())
	ok := func(c *gin.Context) { c.Status(http.StatusNoContent) }
	api.GET("/boards", ok)
	api.GET("/boards/:id", ok)
	api.POST("/boards", ok)
	api.PUT("/boards/:id/members", ok)
	api.GET("/boards/:id/secrets", ok)
	api.GET("/system/keys", ok)
	api.GET("/post-its/:id/settings", ok)
	return r
}

func TestRestrictAccessTokens(t *testing.T) {
	r := restricted()

	cases := []struct {
		token, method, path string
		want                int
	}{
		{"pat", http.MethodGet, "/api/v1/boards", http.StatusNoContent},
		{"pat", http.MethodGet, "/api/v1/boards/b1", http.StatusNoContent},
		{"pat", http.MethodPost, "/api/v1/boards", http.StatusNoContent},
		{"pat", http.MethodPut, "/api/v1/boards/b1/members", http.StatusForbidden},
		{"pat", http.MethodGet, "/api/v1/boards/b1/secrets", http.StatusForbidden},
		{"pat", http.MethodGet, "/api/v1/system/keys", http.StatusForbidden},
		{"pat", http.MethodGet, "/api/v1/post-its/p1/settings", http.StatusForbidden},
		// A browser session is not restricted.
		{"session", http.MethodPost, "/api/v1/boards", http.StatusNoContent},
		{"session", http.MethodGet, "/api/v1/boards/b1/secrets", http.StatusNoContent},
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.path, nil)
		req.Header.Set("Authorization", "Bearer "+c.token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != c.want {
			t.Errorf("%s %s %s: %d, want %d", c.token, c.method, c.path, w.Code, c.want)
		}
	}
}

func TestRequireAuth_MarksAccessTokenPrincipals(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/who", RequireAuth(mocks.StaticVerifier{"pat": infrastructure.Claims{Subject: "u", AccessToken: true}}), func(c *gin.Context) {
		if p := Principal(c); !p.Token || p.ID != "u" || p.Admin {
			t.Errorf("principal = %+v", p)
		}
		c.Status(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/who", nil)
	req.Header.Set("Authorization", "Bearer pat")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d", w.Code)
	}
}

func TestPlanLimit_AnswersPaymentRequired(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/limit", func(c *gin.Context) {
		if !PlanLimit(c, &models.LimitError{Limit: models.LimitBoards, Max: 5, Plan: models.PlanFree}) {
			t.Error("a LimitError was not recognised")
		}
	})
	r.GET("/wrapped", func(c *gin.Context) {
		PlanLimit(c, fmt.Errorf("create: %w", &models.LimitError{Limit: models.LimitAPIAccess, Plan: models.PlanFree}))
	})
	r.GET("/other", func(c *gin.Context) {
		if PlanLimit(c, errors.New("boom")) {
			t.Error("a plain error was taken for a plan limit")
		}
	})

	for path, want := range map[string]string{
		"/limit":   `{"error":"plan_limit","limit":"boards","max":5,"plan":"free"}`,
		"/wrapped": `{"error":"plan_limit","limit":"api_access","max":0,"plan":"free"}`,
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusPaymentRequired || w.Body.String() != want {
			t.Errorf("%s: %d %s", path, w.Code, w.Body.String())
		}
	}
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/other", nil))
}

func TestAccessTokenAllowlist_NeverOpensBillingOrSystem(t *testing.T) {
	for route := range accessTokenRoutes {
		if strings.Contains(route, "/billing") || strings.Contains(route, "/system") {
			t.Errorf("%s is open to access tokens", route)
		}
	}
	if !accessTokenRoutes["DELETE /api/v1/boards/:id/strands/:strand"] {
		t.Error("tokens cannot remove a strand")
	}
}

func TestRequireAuth_AccessTokenWithoutAPIAccessIsPaymentRequired(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequireAuth(failingVerifier{&models.LimitError{Limit: models.LimitAPIAccess, Plan: models.PlanFree}}))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer tiu_pat_x")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusPaymentRequired || !strings.Contains(w.Body.String(), `"limit":"api_access"`) {
		t.Errorf("%d %s", w.Code, w.Body.String())
	}
}

type failingVerifier struct{ err error }

func (f failingVerifier) Verify(string) (infrastructure.Claims, error) {
	return infrastructure.Claims{}, f.err
}
