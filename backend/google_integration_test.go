//go:build integration

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	a_ctrl "github.com/Secreto31126/tesis/common/controllers/auth"
	a_srv "github.com/Secreto31126/tesis/common/services/auth"
)

// The mock-oauth2-server of the integration profile plays Google: its login
// form takes a username (the subject) and the extra id_token claims as JSON.
const (
	googleIssuer       = "http://localhost:8899/default"
	googleClientID     = "tiu-integration"
	googleClientSecret = "tiu-integration-secret"
)

type googleAccount struct {
	subject       string
	email         string
	emailVerified bool
	name          string
}

type googleCallback struct {
	code, state string
	binding     *http.Cookie
}

// consentAtGoogle starts a sign-in, logs into the mock provider as the given
// account and returns what the provider handed back to the callback page.
func (s *stack) consentAtGoogle(account googleAccount, next string) googleCallback {
	s.t.Helper()

	w := s.do(http.MethodGet, "/api/v1/auth/google/start?next="+url.QueryEscape(next), nil)
	var start a_ctrl.GoogleStartResponse
	s.mustJSON(w, http.StatusOK, &start)
	var binding *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == a_ctrl.SIGNIN_COOKIE {
			binding = c
		}
	}
	if binding == nil {
		s.t.Fatal("start did not set the binding cookie")
	}

	claims, _ := json.Marshal(map[string]any{"email": account.email, "email_verified": account.emailVerified, "name": account.name})
	browser := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := browser.PostForm(start.AuthorizationURL, url.Values{"username": {account.subject}, "claims": {string(claims)}})
	if err != nil {
		s.t.Fatalf("consent: %v", err)
	}
	defer res.Body.Close()

	location, err := url.Parse(res.Header.Get("Location"))
	if err != nil || !strings.HasSuffix(location.Path, a_srv.GOOGLE_CALLBACK_PATH) {
		s.t.Fatalf("provider redirected to %q (status %d)", res.Header.Get("Location"), res.StatusCode)
	}
	return googleCallback{code: location.Query().Get("code"), state: location.Query().Get("state"), binding: binding}
}

func (s *stack) finishGoogle(callback googleCallback) *httptest.ResponseRecorder {
	s.t.Helper()
	body, _ := json.Marshal(map[string]string{"code": callback.code, "state": callback.state})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/google/callback", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	if callback.binding != nil {
		req.AddCookie(callback.binding)
	}
	w := httptest.NewRecorder()
	s.app.router.ServeHTTP(w, req)
	s.responses = append(s.responses, w)
	return w
}

func (s *stack) signInWithGoogle(account googleAccount) a_ctrl.GoogleSessionResponse {
	s.t.Helper()
	var session a_ctrl.GoogleSessionResponse
	s.mustJSON(s.finishGoogle(s.consentAtGoogle(account, "/")), http.StatusOK, &session)
	return session
}

func TestEndToEnd_GoogleSignIn(t *testing.T) {
	s := newStack(t)
	ana := googleAccount{subject: "google-ana", email: "Ana.Google@it.test", emailVerified: true, name: "Ana Google"}

	callback := s.consentAtGoogle(ana, "/board/42")
	if callback.code == "" || callback.state == "" {
		t.Fatalf("callback = %+v", callback)
	}

	stolen := callback
	stolen.binding = nil
	s.mustJSON(s.finishGoogle(stolen), http.StatusBadRequest, nil)

	var session a_ctrl.GoogleSessionResponse
	w := s.finishGoogle(callback)
	s.mustJSON(w, http.StatusOK, &session)
	if session.Next != "/board/42" || session.User.Email != "ana.google@it.test" || !session.User.EmailVerified || session.User.Name != "Ana Google" {
		t.Errorf("session = %+v", session)
	}
	if cookie := refreshCookie(w); cookie == nil || !cookie.HttpOnly {
		t.Fatalf("refresh cookie = %+v", cookie)
	}

	w = s.bearer(session.AccessToken, http.MethodGet, "/api/v1/users/"+session.User.Id.String())
	s.mustJSON(w, http.StatusOK, nil)
	if !strings.Contains(w.Body.String(), `"identities":["google"]`) {
		t.Errorf("profile = %s", w.Body.String())
	}

	s.mustJSON(s.finishGoogle(callback), http.StatusBadRequest, nil)

	ana.email = "ana.renamed@it.test"
	again := s.signInWithGoogle(ana)
	if again.User.Id != session.User.Id {
		t.Errorf("the same google subject must reach the same account: %v vs %v", again.User.Id, session.User.Id)
	}

	s.assertNoResponseContains(googleClientSecret)
	s.assertNoResponseContains(callback.code)
	s.assertNoResponseContains("id_token")
}

func TestEndToEnd_PasswordAccountLinksGoogle(t *testing.T) {
	s := newStack(t)
	const secret = "a password for bob"

	s.mustJSON(s.do(http.MethodPost, "/api/v1/auth/register", map[string]string{"email": "bob@it.test", "password": secret, "name": "Bob"}), http.StatusAccepted, nil)
	var registered a_ctrl.SessionResponse
	s.mustJSON(s.do(http.MethodPost, "/api/v1/auth/verify-email", map[string]string{"token": s.lastMailedToken()}), http.StatusOK, &registered)

	session := s.signInWithGoogle(googleAccount{subject: "google-bob", email: "bob@it.test", emailVerified: true, name: "Bob G"})

	if session.User.Id != registered.User.Id || session.User.Name != "Bob" {
		t.Errorf("google must join the existing account: %+v", session.User)
	}
	w := s.bearer(session.AccessToken, http.MethodGet, "/api/v1/users/"+session.User.Id.String())
	s.mustJSON(w, http.StatusOK, nil)
	if !strings.Contains(w.Body.String(), `"identities":["password","google"]`) {
		t.Errorf("profile = %s", w.Body.String())
	}
	s.mustJSON(s.do(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "bob@it.test", "password": secret}), http.StatusOK, nil)
}

func TestEndToEnd_GoogleRefusesAnUnverifiedEmail(t *testing.T) {
	s := newStack(t)

	callback := s.consentAtGoogle(googleAccount{subject: "google-carol", email: "carol@it.test", name: "Carol"}, "/")
	s.mustJSON(s.finishGoogle(callback), http.StatusForbidden, nil)

	s.mustJSON(s.do(http.MethodPost, "/api/v1/auth/password/forgot", map[string]string{"email": "carol@it.test"}), http.StatusAccepted, nil)
	if _, sent := s.mailer.Last(); sent {
		t.Error("a refused google sign-in must not create an account")
	}
}
