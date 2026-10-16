//go:build integration

package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	a_ctrl "github.com/Secreto31126/tesis/common/controllers/auth"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

// fromSite sends the refresh cookie the way a browser labels where the request
// comes from.
func (s *stack) fromSite(site string, cookie *http.Cookie, method, path string) *httptest.ResponseRecorder {
	s.t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.AddCookie(cookie)
	req.Header.Set("Sec-Fetch-Site", site)
	w := httptest.NewRecorder()
	s.app.router.ServeHTTP(w, req)
	s.responses = append(s.responses, w)
	return w
}

// verified registers an account and answers the cookie its verification set.
func (s *stack) verified() *http.Cookie {
	s.t.Helper()
	email := "cookie-" + uuid.NewString()[:8] + "@it.test"
	s.mustJSON(s.requestFrom(freshAddress(), "", http.MethodPost, "/api/v1/auth/register",
		map[string]string{"email": email, "password": itPassword, "name": "Cookie"}), http.StatusAccepted, nil)

	w := s.do(http.MethodPost, "/api/v1/auth/verify-email", map[string]string{"token": s.lastMailedToken()})
	s.mustJSON(w, http.StatusOK, nil)
	cookie := refreshCookie(w)
	if cookie == nil {
		s.t.Fatal("verification set no refresh cookie")
	}
	return cookie
}

func TestEndToEnd_RefreshCookieAttributes(t *testing.T) {
	s := newStack(t)
	cookie := s.verified()

	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("cookie flags = %+v, want HttpOnly, Secure and SameSite=Strict", cookie)
	}
	if cookie.Path != AUTH_COOKIE_PATH || cookie.MaxAge != int(DEFAULT_REFRESH_TTL.Seconds()) {
		t.Errorf("cookie path %q, max age %d", cookie.Path, cookie.MaxAge)
	}
}

func TestEndToEnd_CookieRoutesRefuseCrossSiteRequests(t *testing.T) {
	s := newStack(t)
	cookie := s.verified()

	for _, path := range []string{"/api/v1/auth/refresh", "/api/v1/auth/logout"} {
		for _, site := range []string{"cross-site", "same-site"} {
			w := s.fromSite(site, cookie, http.MethodPost, path)
			if w.Code != http.StatusForbidden || refreshCookie(w) != nil {
				t.Errorf("%s from %s: %d, cookie %+v, want 403 and no cookie", path, site, w.Code, refreshCookie(w))
			}
		}
	}

	// The refused requests spent nothing: the cookie still renews.
	w := s.fromSite("same-origin", cookie, http.MethodPost, "/api/v1/auth/refresh")
	s.mustJSON(w, http.StatusOK, nil)
}

func TestEndToEnd_LoginIsRateLimited(t *testing.T) {
	s := newStack(t)
	ana := s.signUp("ana")
	address := freshAddress()
	login := func(from, password string) *httptest.ResponseRecorder {
		return s.requestFrom(from, "", http.MethodPost, "/api/v1/auth/login", map[string]string{"email": ana.email, "password": password})
	}

	for i := 0; i < 10; i++ {
		if w := login(address, "wrong password"); w.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: %d, want 401", i+1, w.Code)
		}
	}
	if w := login(address, itPassword); w.Code != http.StatusTooManyRequests || !strings.Contains(w.Body.String(), "rate_limited") {
		t.Errorf("eleventh attempt: %d %s, want 429 rate_limited", w.Code, w.Body.String())
	}
	if w := login(freshAddress(), itPassword); w.Code != http.StatusTooManyRequests {
		t.Errorf("the same account from another address: %d, want 429", w.Code)
	}

	var session a_ctrl.SessionResponse
	bob := s.signUp("bob")
	s.mustJSON(s.requestFrom(freshAddress(), "", http.MethodPost, "/api/v1/auth/login",
		map[string]string{"email": bob.email, "password": itPassword}), http.StatusOK, &session)
}

// Adding people by email tells whether they have an account, so boards and
// organizations share one allowance of lookups per user.
func TestEndToEnd_MemberLookupsAreRateLimited(t *testing.T) {
	s := newStack(t)
	owner := s.signUp("owner")
	board := owner.newBoard("Lookups")
	members := "/api/v1/boards/" + board.Id.String() + "/members"

	for i := 0; i < 30; i++ {
		w := owner.do(http.MethodPut, members, map[string]any{"email": "nobody-" + uuid.NewString()[:8] + "@it.test", "role": models.BoardViewer})
		if w.Code != http.StatusNotFound {
			t.Fatalf("lookup %d: %d %s, want 404", i+1, w.Code, w.Body.String())
		}
	}

	viewer := s.signUp("viewer")
	if w := owner.do(http.MethodPut, members, map[string]any{"email": viewer.email, "role": models.BoardViewer}); w.Code != http.StatusTooManyRequests {
		t.Errorf("past the limit on a board: %d %s, want 429", w.Code, w.Body.String())
	}

	var org models.Org
	s.mustJSON(owner.do(http.MethodPost, "/api/v1/orgs", map[string]string{"name": "Lookups"}), http.StatusCreated, &org)
	if w := owner.do(http.MethodPut, "/api/v1/orgs/"+org.Id.String()+"/members", map[string]any{"email": viewer.email, "role": models.OrgRoleMember}); w.Code != http.StatusTooManyRequests {
		t.Errorf("past the limit on an organization: %d %s, want 429", w.Code, w.Body.String())
	}

	if w := viewer.do(http.MethodPost, "/api/v1/boards", map[string]string{"name": "Mine"}); w.Code != http.StatusCreated {
		t.Errorf("another user is not limited: %d", w.Code)
	}
}
