//go:build integration

package main

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	a_ctrl "github.com/Secreto31126/tesis/common/controllers/auth"
)

var mailedToken = regexp.MustCompile(`token=([A-Za-z0-9_-]+)`)

func (s *stack) lastMailedToken() string {
	s.t.Helper()
	mail, ok := s.mailer.Last()
	if !ok {
		s.t.Fatal("no mail was sent")
	}
	m := mailedToken.FindStringSubmatch(mail.Text)
	if m == nil {
		s.t.Fatalf("no token in %q", mail.Text)
	}
	return m[1]
}

func refreshCookie(w *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range w.Result().Cookies() {
		if c.Name == a_ctrl.REFRESH_COOKIE {
			return c
		}
	}
	return nil
}

func (s *stack) withCookie(cookie *http.Cookie, method, path string) *httptest.ResponseRecorder {
	s.t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	s.app.router.ServeHTTP(w, req)
	s.responses = append(s.responses, w)
	return w
}

func (s *stack) bearer(token, method, path string) *httptest.ResponseRecorder {
	s.t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.app.router.ServeHTTP(w, req)
	s.responses = append(s.responses, w)
	return w
}

func TestEndToEnd_PasswordSignupLoginRefreshLogout(t *testing.T) {
	s := newStack(t)
	const secret = "correct horse battery staple"

	w := s.do(http.MethodPost, "/api/v1/auth/register", map[string]string{"email": "ana@it.test", "password": secret, "name": "Ana"})
	s.mustJSON(w, http.StatusAccepted, nil)
	verifyToken := s.lastMailedToken()

	w = s.do(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "ana@it.test", "password": secret})
	s.mustJSON(w, http.StatusForbidden, nil)

	var session a_ctrl.SessionResponse
	w = s.do(http.MethodPost, "/api/v1/auth/verify-email", map[string]string{"token": verifyToken})
	s.mustJSON(w, http.StatusOK, &session)
	if !session.User.EmailVerified || session.User.Admin || session.User.Email != "ana@it.test" {
		t.Errorf("user = %+v", session.User)
	}
	first := refreshCookie(w)
	if first == nil || !first.HttpOnly || first.Path != AUTH_COOKIE_PATH {
		t.Fatalf("cookie = %+v", first)
	}

	w = s.bearer(session.AccessToken, http.MethodGet, "/api/v1/users/"+session.User.Id.String())
	s.mustJSON(w, http.StatusOK, nil)
	if !strings.Contains(w.Body.String(), `"identities":["password"]`) {
		t.Errorf("profile = %s", w.Body.String())
	}
	if w := s.bearer("forged", http.MethodGet, "/api/v1/users/"+session.User.Id.String()); w.Code != http.StatusUnauthorized {
		t.Errorf("forged token: %d", w.Code)
	}

	var rotated a_ctrl.SessionResponse
	w = s.withCookie(first, http.MethodPost, "/api/v1/auth/refresh")
	s.mustJSON(w, http.StatusOK, &rotated)
	second := refreshCookie(w)
	if second == nil || second.Value == first.Value || rotated.AccessToken == "" {
		t.Fatalf("refresh did not rotate: %+v", second)
	}
	if w := s.withCookie(first, http.MethodPost, "/api/v1/auth/refresh"); w.Code != http.StatusUnauthorized {
		t.Errorf("spent cookie: %d", w.Code)
	}

	w = s.withCookie(second, http.MethodPost, "/api/v1/auth/logout")
	if w.Code != http.StatusNoContent || refreshCookie(w).MaxAge >= 0 {
		t.Errorf("logout: %d, cookie = %+v", w.Code, refreshCookie(w))
	}
	if w := s.withCookie(second, http.MethodPost, "/api/v1/auth/refresh"); w.Code != http.StatusUnauthorized {
		t.Errorf("refresh after logout: %d", w.Code)
	}

	w = s.do(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "ana@it.test", "password": secret})
	s.mustJSON(w, http.StatusOK, &session)

	s.assertNoResponseContains(secret)
	s.assertNoResponseContains(first.Value)
	s.assertNoResponseContains(second.Value)
	s.assertNoResponseContains("$argon2id$")
}

func TestEndToEnd_ResetLogsEverySessionOutAndAdminsAreSeeded(t *testing.T) {
	s := newStack(t)
	const secret = "first password!"

	s.mustJSON(s.do(http.MethodPost, "/api/v1/auth/register", map[string]string{"email": "admin@it.test", "password": secret, "name": "Root"}), http.StatusAccepted, nil)
	var session a_ctrl.SessionResponse
	s.mustJSON(s.do(http.MethodPost, "/api/v1/auth/verify-email", map[string]string{"token": s.lastMailedToken()}), http.StatusOK, &session)
	if !session.User.Admin {
		t.Error("PLATFORM_ADMINS must make the user an admin")
	}
	cookie := refreshCookie(s.responses[len(s.responses)-1])

	s.mustJSON(s.do(http.MethodPost, "/api/v1/auth/password/forgot", map[string]string{"email": "admin@it.test"}), http.StatusAccepted, nil)
	var fresh a_ctrl.SessionResponse
	s.mustJSON(s.do(http.MethodPost, "/api/v1/auth/password/reset", map[string]string{"token": s.lastMailedToken(), "password": "second password!"}), http.StatusOK, &fresh)

	if w := s.withCookie(cookie, http.MethodPost, "/api/v1/auth/refresh"); w.Code != http.StatusUnauthorized {
		t.Errorf("old session after reset: %d", w.Code)
	}
	s.mustJSON(s.do(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "admin@it.test", "password": secret}), http.StatusUnauthorized, nil)
	s.mustJSON(s.do(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "admin@it.test", "password": "second password!"}), http.StatusOK, nil)
}
