package middleware

import (
	"net/http"
	"net/http/httptest"
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
		{"pat", http.MethodPost, "/api/v1/boards", http.StatusForbidden},
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
