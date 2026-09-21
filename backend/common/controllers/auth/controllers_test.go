package auth

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/mocks"
	srv "github.com/Secreto31126/tesis/common/services/auth"
	"github.com/gin-gonic/gin"
)

const password = "correct horse battery"

type signer struct{}

func (signer) Sign(c infrastructure.Claims) (string, error) { return "access:" + c.Subject, nil }

type harness struct {
	r       *gin.Engine
	mailer  *mocks.RecordingMailer
	limiter *mocks.CountingLimiter
	users   *mocks.MemoryUserStore
}

func setup(t *testing.T, secure bool) *harness {
	t.Helper()
	h := &harness{mailer: &mocks.RecordingMailer{}, limiter: &mocks.CountingLimiter{}, users: &mocks.MemoryUserStore{}}
	service := srv.New(srv.Config{AccessTTL: 15 * time.Minute, RefreshTTL: 30 * 24 * time.Hour},
		h.users, &mocks.MemorySessionStore{}, mocks.PlainHasher{}, signer{}, &mocks.MockHandshakeStore{}, h.limiter, h.mailer)

	gin.SetMode(gin.TestMode)
	h.r = gin.New()
	api := h.r.Group("/api/v1")
	NewController(service, Cookies{Secure: secure, Path: "/api/v1/auth"}).RegisterRoutes(api)
	return h
}

type response struct {
	*httptest.ResponseRecorder
	cookie *http.Cookie
}

func (h *harness) do(method, path, body string, headers ...string) response {
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Host = "tiu.example"
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	w := httptest.NewRecorder()
	h.r.ServeHTTP(w, req)

	res := response{ResponseRecorder: w}
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == REFRESH_COOKIE {
			res.cookie = cookie
		}
	}
	return res
}

var linkToken = regexp.MustCompile(`token=([A-Za-z0-9_-]+)`)

func (h *harness) lastToken(t *testing.T) string {
	t.Helper()
	mail, ok := h.mailer.Last()
	if !ok {
		t.Fatal("no mail")
	}
	return linkToken.FindStringSubmatch(mail.Text)[1]
}

func (h *harness) signUp(t *testing.T, email string) response {
	t.Helper()
	if w := h.do(http.MethodPost, "/api/v1/auth/register", `{"email":"`+email+`","password":"`+password+`","name":"Ana"}`); w.Code != http.StatusAccepted {
		t.Fatalf("register: %d %s", w.Code, w.Body.String())
	}
	w := h.do(http.MethodPost, "/api/v1/auth/verify-email", `{"token":"`+h.lastToken(t)+`"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("verify: %d %s", w.Code, w.Body.String())
	}
	return w
}

func withCookie(cookie *http.Cookie) []string {
	return []string{"Cookie", cookie.Name + "=" + cookie.Value}
}

func TestRegister_AcceptsAndMailsWithTheRequestOrigin(t *testing.T) {
	h := setup(t, true)

	w := h.do(http.MethodPost, "/api/v1/auth/register", `{"email":"ana@example.com","password":"`+password+`","name":"Ana"}`)
	if w.Code != http.StatusAccepted || w.Body.String() != "{}" {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
	mail, _ := h.mailer.Last()
	if !strings.Contains(mail.Text, "https://tiu.example/verify?token=") {
		t.Errorf("mail = %q", mail.Text)
	}
	if w.cookie != nil {
		t.Error("register must not start a session")
	}
}

func TestRegister_ValidationErrors(t *testing.T) {
	h := setup(t, true)

	cases := map[string]string{
		`{"email":"nope","password":"` + password + `","name":"Ana"}`: "invalid email",
		`{"email":"a@b.co","password":"short","name":"Ana"}`:          "password must have between 8 and 128 characters",
		`{"email":"a@b.co","password":"` + password + `","name":" "}`: "invalid name",
		`{"password":"` + password + `","name":"Ana"}`:                "",
		`not json`: "",
	}
	for body, want := range cases {
		w := h.do(http.MethodPost, "/api/v1/auth/register", body)
		if w.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d", body, w.Code)
		}
		if want != "" && !strings.Contains(w.Body.String(), want) {
			t.Errorf("%s: body = %s", body, w.Body.String())
		}
	}
}

func TestVerify_ReturnsTokensAndSetsTheRefreshCookie(t *testing.T) {
	h := setup(t, true)
	w := h.signUp(t, "ana@example.com")

	var body struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
		User        struct {
			Email         string   `json:"email"`
			EmailVerified bool     `json:"email_verified"`
			Identities    []string `json:"identities"`
		} `json:"user"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(body.AccessToken, "access:") || body.TokenType != "Bearer" || body.ExpiresIn != 900 {
		t.Errorf("body = %+v", body)
	}
	if body.User.Email != "ana@example.com" || !body.User.EmailVerified || len(body.User.Identities) != 1 {
		t.Errorf("user = %+v", body.User)
	}
	if strings.Contains(w.Body.String(), "refresh") || strings.Contains(w.Body.String(), "plain:") {
		t.Errorf("body leaks session or hash material: %s", w.Body.String())
	}

	c := w.cookie
	if c == nil {
		t.Fatal("no refresh cookie")
	}
	if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteStrictMode || c.Path != "/api/v1/auth" || c.MaxAge != 30*24*3600 || c.Value == "" {
		t.Errorf("cookie = %+v", c)
	}
}

func TestCookies_SecureFlagFollowsConfig(t *testing.T) {
	h := setup(t, false)
	w := h.signUp(t, "ana@example.com")
	if w.cookie.Secure {
		t.Error("AUTH_COOKIE_SECURE=false must yield a non-secure cookie")
	}
}

func TestLogin_StatusCodes(t *testing.T) {
	h := setup(t, true)
	h.signUp(t, "ana@example.com")
	_ = h.do(http.MethodPost, "/api/v1/auth/register", `{"email":"new@example.com","password":"`+password+`","name":"New"}`)

	cases := []struct {
		body string
		want int
		err  string
	}{
		{`{"email":"ana@example.com","password":"` + password + `"}`, http.StatusOK, ""},
		{`{"email":"ana@example.com","password":"wrong"}`, http.StatusUnauthorized, "invalid_credentials"},
		{`{"email":"nobody@example.com","password":"` + password + `"}`, http.StatusUnauthorized, "invalid_credentials"},
		{`{"email":"new@example.com","password":"` + password + `"}`, http.StatusForbidden, "email_not_verified"},
		{`{"email":"ana@example.com"}`, http.StatusBadRequest, ""},
	}
	for _, c := range cases {
		w := h.do(http.MethodPost, "/api/v1/auth/login", c.body)
		if w.Code != c.want {
			t.Errorf("%s: status = %d, want %d (%s)", c.body, w.Code, c.want, w.Body.String())
		}
		if c.err != "" && !strings.Contains(w.Body.String(), `"error":"`+c.err+`"`) {
			t.Errorf("%s: body = %s", c.body, w.Body.String())
		}
		if c.want == http.StatusOK && w.cookie == nil {
			t.Error("login must set the refresh cookie")
		}
		if c.want != http.StatusOK && w.cookie != nil {
			t.Error("failed login must not touch the cookie")
		}
	}
}

func TestLogin_RateLimited(t *testing.T) {
	h := setup(t, true)
	h.limiter.Refuse = map[string]bool{"login:ana@example.com": true}

	w := h.do(http.MethodPost, "/api/v1/auth/login", `{"email":"ana@example.com","password":"x"}`)
	if w.Code != http.StatusTooManyRequests || !strings.Contains(w.Body.String(), "rate_limited") {
		t.Errorf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestRefresh_RotatesTheCookieAndRejectsWithoutIt(t *testing.T) {
	h := setup(t, true)
	first := h.signUp(t, "ana@example.com")

	w := h.do(http.MethodPost, "/api/v1/auth/refresh", "", withCookie(first.cookie)...)
	if w.Code != http.StatusOK || w.cookie == nil || w.cookie.Value == first.cookie.Value {
		t.Fatalf("refresh: %d, cookie = %+v", w.Code, w.cookie)
	}
	if !strings.Contains(w.Body.String(), `"access_token"`) {
		t.Errorf("body = %s", w.Body.String())
	}

	w = h.do(http.MethodPost, "/api/v1/auth/refresh", "")
	if w.Code != http.StatusUnauthorized || w.cookie == nil || w.cookie.MaxAge >= 0 {
		t.Errorf("no cookie: status = %d, cookie = %+v (must be cleared)", w.Code, w.cookie)
	}

	w = h.do(http.MethodPost, "/api/v1/auth/refresh", "", withCookie(&http.Cookie{Name: REFRESH_COOKIE, Value: "garbage"})...)
	if w.Code != http.StatusUnauthorized || w.cookie == nil || w.cookie.MaxAge >= 0 {
		t.Errorf("bad cookie: status = %d, cookie = %+v", w.Code, w.cookie)
	}
}

func TestRefreshAndLogout_RefuseCrossSiteRequests(t *testing.T) {
	h := setup(t, true)
	first := h.signUp(t, "ana@example.com")

	for _, path := range []string{"/api/v1/auth/refresh", "/api/v1/auth/logout"} {
		headers := append(withCookie(first.cookie), "Sec-Fetch-Site", "cross-site")
		if w := h.do(http.MethodPost, path, "", headers...); w.Code != http.StatusForbidden {
			t.Errorf("%s cross-site: status = %d", path, w.Code)
		}
	}
	if w := h.do(http.MethodPost, "/api/v1/auth/refresh", "", withCookie(first.cookie)...); w.Code != http.StatusOK {
		t.Errorf("the session must survive a refused request: %d", w.Code)
	}
}

func TestLogout_ClearsTheCookieAndKillsTheSession(t *testing.T) {
	h := setup(t, true)
	first := h.signUp(t, "ana@example.com")

	w := h.do(http.MethodPost, "/api/v1/auth/logout", "", withCookie(first.cookie)...)
	if w.Code != http.StatusNoContent || w.cookie == nil || w.cookie.MaxAge >= 0 {
		t.Errorf("logout: status = %d, cookie = %+v", w.Code, w.cookie)
	}
	if w := h.do(http.MethodPost, "/api/v1/auth/refresh", "", withCookie(first.cookie)...); w.Code != http.StatusUnauthorized {
		t.Errorf("refresh after logout: %d", w.Code)
	}
	if w := h.do(http.MethodPost, "/api/v1/auth/logout", ""); w.Code != http.StatusNoContent {
		t.Errorf("logout without cookie: %d", w.Code)
	}
}

func TestForgotAndReset(t *testing.T) {
	h := setup(t, true)
	h.signUp(t, "ana@example.com")

	if w := h.do(http.MethodPost, "/api/v1/auth/password/forgot", `{"email":"nobody@example.com"}`); w.Code != http.StatusAccepted {
		t.Errorf("unknown: %d", w.Code)
	}
	if w := h.do(http.MethodPost, "/api/v1/auth/password/forgot", `{"email":"ana@example.com"}`); w.Code != http.StatusAccepted {
		t.Errorf("known: %d", w.Code)
	}
	token := h.lastToken(t)

	if w := h.do(http.MethodPost, "/api/v1/auth/password/reset", `{"token":"`+token+`","password":"short"}`); w.Code != http.StatusBadRequest {
		t.Errorf("weak: %d", w.Code)
	}
	w := h.do(http.MethodPost, "/api/v1/auth/password/reset", `{"token":"`+token+`","password":"brand new password"}`)
	if w.Code != http.StatusOK || w.cookie == nil {
		t.Errorf("reset: %d %s", w.Code, w.Body.String())
	}
	if w := h.do(http.MethodPost, "/api/v1/auth/password/reset", `{"token":"`+token+`","password":"brand new password"}`); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "invalid_token") {
		t.Errorf("reused: %d %s", w.Code, w.Body.String())
	}
	if w := h.do(http.MethodPost, "/api/v1/auth/verify-email", `{"token":"nope"}`); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "invalid_token") {
		t.Errorf("bad verify: %d %s", w.Code, w.Body.String())
	}
}
