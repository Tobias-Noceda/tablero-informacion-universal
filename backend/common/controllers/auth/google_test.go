package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
)

func verifiedGoogleUser() *infrastructure.ExternalIdentity {
	return &infrastructure.ExternalIdentity{Provider: models.IdentityGoogle, Subject: "sub-1", Email: "ana@example.com", EmailVerified: true, Name: "Ana"}
}

func (h *harness) startGoogle(t *testing.T, next string) (state string, signin *http.Cookie) {
	t.Helper()
	w := h.do(http.MethodGet, "/api/v1/auth/google/start?next="+url.QueryEscape(next), "")
	if w.Code != http.StatusOK {
		t.Fatalf("start: %d %s", w.Code, w.Body.String())
	}
	var body struct {
		AuthorizationURL string `json:"authorization_url"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	parsed, err := url.Parse(body.AuthorizationURL)
	if err != nil {
		t.Fatalf("authorization_url = %q", body.AuthorizationURL)
	}
	return parsed.Query().Get("state"), w.signin
}

func (h *harness) finishGoogle(state string, signin *http.Cookie, headers ...string) response {
	if signin != nil {
		headers = append(headers, withCookie(signin)...)
	}
	return h.do(http.MethodPost, "/api/v1/auth/google/callback", `{"code":"the-code","state":"`+state+`"}`, headers...)
}

func TestGoogleStart_AnswersTheConsentURLAndBindsTheBrowser(t *testing.T) {
	h := setup(t, true)

	w := h.do(http.MethodGet, "/api/v1/auth/google/start?next=/board/1", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d %s", w.Code, w.Body.String())
	}
	var body struct {
		AuthorizationURL string `json:"authorization_url"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	parsed, _ := url.Parse(body.AuthorizationURL)
	if got := parsed.Query().Get("redirect_uri"); got != "https://tiu.example/auth/google/callback" {
		t.Errorf("redirect_uri = %q", got)
	}

	c := w.signin
	if c == nil {
		t.Fatal("no sign-in binding cookie")
	}
	sum := sha256.Sum256([]byte(parsed.Query().Get("state")))
	if c.Value != hex.EncodeToString(sum[:]) {
		t.Error("the binding cookie must hold the hash of the state")
	}
	if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode || c.Path != "/api/v1/auth/google" || c.MaxAge != 600 {
		t.Errorf("cookie = %+v", c)
	}
	if w.cookie != nil {
		t.Error("starting a sign-in must not touch the session cookie")
	}
}

func TestGoogleStart_RefusesAnOffsiteNext(t *testing.T) {
	h := setup(t, true)

	w := h.do(http.MethodGet, "/api/v1/auth/google/start?next="+url.QueryEscape("https://evil.example"), "")
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "invalid_next") || w.signin != nil {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestGoogle_NotConfigured(t *testing.T) {
	h := setup(t, true)
	h.mount(true, nil)

	if w := h.do(http.MethodGet, "/api/v1/auth/google/start", ""); w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "google_not_configured") {
		t.Errorf("start: %d %s", w.Code, w.Body.String())
	}
	if w := h.finishGoogle("state", nil); w.Code != http.StatusServiceUnavailable {
		t.Errorf("callback: %d %s", w.Code, w.Body.String())
	}
}

func TestGoogleCallback_StartsASessionAndForgetsTheBinding(t *testing.T) {
	h := setup(t, true)
	h.google.Identity = verifiedGoogleUser()
	state, signin := h.startGoogle(t, "/board/1")

	w := h.finishGoogle(state, signin)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d %s", w.Code, w.Body.String())
	}
	var body struct {
		AccessToken string `json:"access_token"`
		Next        string `json:"next"`
		User        struct {
			Email      string   `json:"email"`
			Identities []string `json:"identities"`
		} `json:"user"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if !strings.HasPrefix(body.AccessToken, "access:") || body.Next != "/board/1" || body.User.Email != "ana@example.com" || body.User.Identities[0] != "google" {
		t.Errorf("body = %s", w.Body.String())
	}
	if w.cookie == nil || !w.cookie.HttpOnly || w.cookie.Path != "/api/v1/auth" {
		t.Errorf("refresh cookie = %+v", w.cookie)
	}
	if w.signin == nil || w.signin.MaxAge >= 0 || w.signin.Path != "/api/v1/auth/google" {
		t.Errorf("the binding cookie must be cleared: %+v", w.signin)
	}
	if strings.Contains(w.Body.String(), state) {
		t.Error("the response must not echo the state")
	}
}

func TestGoogleCallback_StatusCodes(t *testing.T) {
	unverified := verifiedGoogleUser()
	unverified.EmailVerified = false

	cases := []struct {
		name     string
		identity *infrastructure.ExternalIdentity
		err      error
		binding  bool
		headers  []string
		want     int
		code     string
	}{
		{"no binding cookie", verifiedGoogleUser(), nil, false, nil, http.StatusBadRequest, "invalid_state"},
		{"provider refused", nil, errors.Join(infrastructure.ErrIdentityProvider, errors.New("invalid_grant")), true, nil, http.StatusBadGateway, "provider_error"},
		{"unverified email", unverified, nil, true, nil, http.StatusForbidden, "email_not_verified"},
		{"cross-site", verifiedGoogleUser(), nil, true, []string{"Sec-Fetch-Site", "cross-site"}, http.StatusForbidden, "forbidden"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := setup(t, true)
			h.google.Identity, h.google.Err = tc.identity, tc.err
			state, signin := h.startGoogle(t, "/")
			if !tc.binding {
				signin = nil
			}

			w := h.finishGoogle(state, signin, tc.headers...)
			if w.Code != tc.want || !strings.Contains(w.Body.String(), `"error":"`+tc.code+`"`) {
				t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
			}
			if w.cookie != nil {
				t.Error("a failed sign-in must not set a session cookie")
			}
		})
	}
}

func TestGoogleCallback_RequiresCodeAndState(t *testing.T) {
	h := setup(t, true)

	if w := h.do(http.MethodPost, "/api/v1/auth/google/callback", `{"state":"s"}`); w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}
