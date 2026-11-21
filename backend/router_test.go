package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// clientIP answers the address the rate limiters would key on.
func clientIP(t *testing.T, proxies []string, remote, forwarded string) string {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := newRouter(config{trustedProxies: proxies})
	router.GET("/ip", func(c *gin.Context) { c.String(http.StatusOK, c.ClientIP()) })

	req := httptest.NewRequest(http.MethodGet, "/ip", nil)
	req.RemoteAddr = remote
	if forwarded != "" {
		req.Header.Set("X-Forwarded-For", forwarded)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w.Body.String()
}

func TestRouter_BehindTheDockerALB(t *testing.T) {
	const alb = "172.18.0.5:41000"

	for name, c := range map[string]struct {
		remote, forwarded, want string
	}{
		"the ALB forwards the client":         {alb, "198.51.100.7", "198.51.100.7"},
		"a header the client made up is left": {alb, "1.2.3.4, 198.51.100.7", "198.51.100.7"},
		"a direct caller cannot claim an IP":  {"198.51.100.9:5000", "1.2.3.4", "198.51.100.9"},
		"no header":                           {"198.51.100.9:5000", "", "198.51.100.9"},
	} {
		t.Run(name, func(t *testing.T) {
			if got := clientIP(t, privateNetworks, c.remote, c.forwarded); got != c.want {
				t.Errorf("client IP = %q, want %q", got, c.want)
			}
		})
	}
}

// Vercel replaces X-Forwarded-For with the caller's address (checked in
// production: a made-up header did not dodge the login limit).
func TestRouter_OnVercel(t *testing.T) {
	if got := clientIP(t, everyNetwork, "10.1.2.3:1234", "203.0.113.5"); got != "203.0.113.5" {
		t.Errorf("client IP = %q, want the address Vercel forwards", got)
	}
}

func TestRouter_TrustingNoProxyUsesTheConnection(t *testing.T) {
	if got := clientIP(t, nil, "10.1.2.3:1234", "203.0.113.5"); got != "10.1.2.3" {
		t.Errorf("client IP = %q, want the connection's address", got)
	}
}
