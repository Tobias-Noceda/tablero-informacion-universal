package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/gin-gonic/gin"
)

func router(verifier infrastructure.TokenVerifier) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/who", RequireAuth(verifier), func(c *gin.Context) {
		p := Principal(c)
		c.JSON(http.StatusOK, gin.H{"id": p.ID, "admin": p.Admin})
	})
	r.GET("/admin", RequireAuth(verifier), RequireAdmin(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	r.POST("/same", SameOrigin(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	return r
}

func get(r http.Handler, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRequireAuth_SetsThePrincipalFromTheBearerToken(t *testing.T) {
	r := router(mocks.StaticVerifier{"good": {Subject: "user-1", Admin: true}})

	w := get(r, "/who", map[string]string{"Authorization": "Bearer good"})
	if w.Code != http.StatusOK || w.Body.String() != `{"admin":true,"id":"user-1"}` {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestRequireAuth_RejectsMissingOrBadTokens(t *testing.T) {
	r := router(mocks.StaticVerifier{"good": {Subject: "user-1"}})

	cases := map[string]map[string]string{
		"no header":     {},
		"wrong scheme":  {"Authorization": "Basic good"},
		"unknown token": {"Authorization": "Bearer bad"},
		"empty bearer":  {"Authorization": "Bearer "},
	}
	for name, headers := range cases {
		w := get(r, "/who", headers)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s: status = %d", name, w.Code)
		}
		if w.Header().Get("WWW-Authenticate") != "Bearer" {
			t.Errorf("%s: WWW-Authenticate = %q", name, w.Header().Get("WWW-Authenticate"))
		}
		if w.Body.String() != `{"error":"unauthorized"}` {
			t.Errorf("%s: body = %s", name, w.Body.String())
		}
	}
}

func TestRequireAdmin_ForbidsNonAdmins(t *testing.T) {
	r := router(mocks.StaticVerifier{"user": {Subject: "u"}, "admin": {Subject: "a", Admin: true}})

	if w := get(r, "/admin", map[string]string{"Authorization": "Bearer user"}); w.Code != http.StatusForbidden || w.Body.String() != `{"error":"forbidden"}` {
		t.Errorf("user: status = %d, body = %s", w.Code, w.Body.String())
	}
	if w := get(r, "/admin", map[string]string{"Authorization": "Bearer admin"}); w.Code != http.StatusNoContent {
		t.Errorf("admin: status = %d", w.Code)
	}
}

func TestSameOrigin_RejectsCrossSiteFetches(t *testing.T) {
	r := router(mocks.StaticVerifier{})

	cases := []struct {
		site string
		want int
	}{
		{"", http.StatusNoContent},
		{"same-origin", http.StatusNoContent},
		{"none", http.StatusNoContent},
		{"cross-site", http.StatusForbidden},
		{"same-site", http.StatusForbidden},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodPost, "/same", nil)
		if c.site != "" {
			req.Header.Set("Sec-Fetch-Site", c.site)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != c.want {
			t.Errorf("Sec-Fetch-Site %q: status = %d, want %d", c.site, w.Code, c.want)
		}
	}
}

func TestPrincipal_IsAnonymousWithoutMiddleware(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if p := Principal(c); !p.Anonymous() {
		t.Errorf("principal = %+v", p)
	}
}
