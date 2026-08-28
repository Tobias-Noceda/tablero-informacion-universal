package oauth

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/ports/safehttp"
)

// A token endpoint listens on loopback, which the SSRF guard rejects by
// design. Relax it only for the test that needs to reach one.
func allowLoopback(t *testing.T) {
	t.Helper()
	original := safehttp.IsSafeIP
	safehttp.IsSafeIP = func(net.IP) bool { return true }
	t.Cleanup(func() { safehttp.IsSafeIP = original })
}

// provider records what the client sent and replies with whatever the test set.
type provider struct {
	server *httptest.Server

	gotForm  url.Values
	gotAuth  string
	gotType  string
	requests int

	status int
	body   string
}

func newProvider(t *testing.T) *provider {
	t.Helper()
	allowLoopback(t)

	p := &provider{status: http.StatusOK}
	p.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.requests++
		_ = r.ParseForm()
		p.gotForm = r.PostForm
		p.gotAuth = r.Header.Get("Authorization")
		p.gotType = r.Header.Get("Content-Type")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(p.status)
		fmt.Fprint(w, p.body)
	}))
	t.Cleanup(p.server.Close)

	return p
}

func (p *provider) material(flow string) *models.OAuth2Material {
	return &models.OAuth2Material{
		Flow:         flow,
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		TokenURL:     p.server.URL,
		Scopes:       "read write",
	}
}

func TestFetch_ClientCredentials(t *testing.T) {
	p := newProvider(t)
	p.body = `{"access_token":"at-1","token_type":"Bearer","expires_in":3600}`

	material := p.material(models.OAuth2ClientCredentials)

	before := time.Now()
	if err := New().Fetch(material); err != nil {
		t.Fatalf("fetch: %v", err)
	}

	if p.gotForm.Get("grant_type") != "client_credentials" {
		t.Errorf("grant_type = %q", p.gotForm.Get("grant_type"))
	}
	if p.gotForm.Get("scope") != "read write" {
		t.Errorf("scope = %q", p.gotForm.Get("scope"))
	}
	if p.gotType != "application/x-www-form-urlencoded" {
		t.Errorf("content type = %q", p.gotType)
	}

	if material.AccessToken != "at-1" || material.TokenType != "Bearer" {
		t.Errorf("material = %+v", material)
	}

	ttl := material.ExpiresAt.Sub(before)
	if ttl < 3590*time.Second || ttl > 3610*time.Second {
		t.Errorf("expires in %v, want ~3600s from the request", ttl)
	}
}

// RFC 6749 requires client_secret_basic, so the credential must travel in the
// Authorization header rather than the form.
func TestFetch_UsesBasicAuthNotFormFields(t *testing.T) {
	p := newProvider(t)
	p.body = `{"access_token":"at","expires_in":60}`

	if err := New().Fetch(p.material(models.OAuth2ClientCredentials)); err != nil {
		t.Fatalf("fetch: %v", err)
	}

	if !strings.HasPrefix(p.gotAuth, "Basic ") {
		t.Fatalf("Authorization = %q, want Basic", p.gotAuth)
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(p.gotAuth, "Basic "))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if string(decoded) != "client-id:client-secret" {
		t.Errorf("basic credentials = %q", decoded)
	}

	if p.gotForm.Get("client_secret") != "" {
		t.Error("the client secret was also sent in the form body")
	}
}

func TestFetch_RefreshTokenGrant(t *testing.T) {
	p := newProvider(t)
	p.body = `{"access_token":"at-2","expires_in":600}`

	material := p.material(models.OAuth2AuthorizationCode)
	material.RefreshToken = "rt-original"

	if err := New().Fetch(material); err != nil {
		t.Fatalf("fetch: %v", err)
	}

	if p.gotForm.Get("grant_type") != "refresh_token" {
		t.Errorf("grant_type = %q, want refresh_token", p.gotForm.Get("grant_type"))
	}
	if p.gotForm.Get("refresh_token") != "rt-original" {
		t.Errorf("refresh_token = %q", p.gotForm.Get("refresh_token"))
	}
}

// A provider that rotates refresh tokens invalidates the old one immediately.
func TestFetch_KeepsRotatedRefreshToken(t *testing.T) {
	p := newProvider(t)
	p.body = `{"access_token":"at","refresh_token":"rt-rotated","expires_in":60}`

	material := p.material(models.OAuth2AuthorizationCode)
	material.RefreshToken = "rt-original"

	if err := New().Fetch(material); err != nil {
		t.Fatalf("fetch: %v", err)
	}

	if material.RefreshToken != "rt-rotated" {
		t.Errorf("refresh token = %q, want the rotated one", material.RefreshToken)
	}
}

// Providers that do not rotate omit the field entirely; the existing token
// must survive or the credential dies after one refresh.
func TestFetch_KeepsOriginalWhenNoRotation(t *testing.T) {
	p := newProvider(t)
	p.body = `{"access_token":"at","expires_in":60}`

	material := p.material(models.OAuth2AuthorizationCode)
	material.RefreshToken = "rt-original"

	if err := New().Fetch(material); err != nil {
		t.Fatalf("fetch: %v", err)
	}

	if material.RefreshToken != "rt-original" {
		t.Errorf("refresh token = %q, want it preserved", material.RefreshToken)
	}
}

func TestFetch_DefaultsTTLWhenExpiresInMissing(t *testing.T) {
	p := newProvider(t)
	p.body = `{"access_token":"at","token_type":"Bearer"}`

	material := p.material(models.OAuth2ClientCredentials)

	before := time.Now()
	if err := New().Fetch(material); err != nil {
		t.Fatalf("fetch: %v", err)
	}

	ttl := material.ExpiresAt.Sub(before)
	if ttl < DEFAULT_TTL-time.Second || ttl > DEFAULT_TTL+time.Second {
		t.Errorf("ttl = %v, want the %v default", ttl, DEFAULT_TTL)
	}
}

// A failed token request commonly echoes the client credentials back in its
// body, so the error must carry the status and nothing else.
func TestFetch_ErrorDoesNotEchoTheBody(t *testing.T) {
	p := newProvider(t)
	p.status = http.StatusUnauthorized
	p.body = `{"error":"invalid_client","client_secret":"client-secret"}`

	err := New().Fetch(p.material(models.OAuth2ClientCredentials))
	if err == nil {
		t.Fatal("expected an error for a non-200 response")
	}
	if strings.Contains(err.Error(), "client-secret") || strings.Contains(err.Error(), "invalid_client") {
		t.Errorf("the error leaked the response body: %v", err)
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error = %v, want it to name the status", err)
	}
}

func TestFetch_RejectsResponseWithoutToken(t *testing.T) {
	p := newProvider(t)
	p.body = `{"token_type":"Bearer","expires_in":60}`

	if err := New().Fetch(p.material(models.OAuth2ClientCredentials)); err == nil {
		t.Error("a response with no access_token was accepted")
	}
}

func TestFetch_RejectsMalformedJSON(t *testing.T) {
	p := newProvider(t)
	p.body = `{not json`

	if err := New().Fetch(p.material(models.OAuth2ClientCredentials)); err == nil {
		t.Error("a malformed response was accepted")
	}
}

func TestFetch_UnauthorizedCredentialMakesNoRequest(t *testing.T) {
	p := newProvider(t)

	// authorization_code with nothing to spend yet.
	material := p.material(models.OAuth2AuthorizationCode)
	if err := New().Fetch(material); err == nil {
		t.Error("expected an error when there is no refresh token")
	}

	material = p.material("implicit")
	if err := New().Fetch(material); err == nil {
		t.Error("expected an error for an unsupported flow")
	}

	if p.requests != 0 {
		t.Errorf("made %d requests, want none before the flow is valid", p.requests)
	}
}

func TestExchange_SendsCodeAndVerifier(t *testing.T) {
	p := newProvider(t)
	p.body = `{"access_token":"at","refresh_token":"rt","token_type":"Bearer","expires_in":3600}`

	material := p.material(models.OAuth2AuthorizationCode)

	if err := New().Exchange(material, "the-code", "https://tablero.example/cb", "the-verifier"); err != nil {
		t.Fatalf("exchange: %v", err)
	}

	if p.gotForm.Get("grant_type") != "authorization_code" {
		t.Errorf("grant_type = %q", p.gotForm.Get("grant_type"))
	}
	if p.gotForm.Get("code") != "the-code" {
		t.Errorf("code = %q", p.gotForm.Get("code"))
	}
	if p.gotForm.Get("redirect_uri") != "https://tablero.example/cb" {
		t.Errorf("redirect_uri = %q", p.gotForm.Get("redirect_uri"))
	}
	if p.gotForm.Get("code_verifier") != "the-verifier" {
		t.Errorf("code_verifier = %q", p.gotForm.Get("code_verifier"))
	}

	if material.AccessToken != "at" || material.RefreshToken != "rt" {
		t.Errorf("material = %+v", material)
	}
}

// Without a refresh token the credential would work once and then break, so
// the handshake is better rejected up front.
func TestExchange_RequiresRefreshToken(t *testing.T) {
	p := newProvider(t)
	p.body = `{"access_token":"at","expires_in":3600}`

	err := New().Exchange(p.material(models.OAuth2AuthorizationCode), "code", "https://x.example/cb", "v")
	if err == nil {
		t.Error("an exchange with no refresh token was accepted")
	}
}

// The guard is the whole reason a user-supplied token URL is safe to fetch.
func TestFetch_SSRFGuardBlocksLoopback(t *testing.T) {
	p := newProvider(t)
	p.body = `{"access_token":"at","expires_in":60}`

	// Undo the relaxation newProvider applied, restoring production behaviour.
	safehttp.IsSafeIP = func(ip net.IP) bool {
		return !(ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified())
	}

	err := New().Fetch(p.material(models.OAuth2ClientCredentials))
	if err == nil {
		t.Fatal("a token endpoint on loopback was reachable")
	}
	if p.requests != 0 {
		t.Error("the request reached a blocked address")
	}
}
