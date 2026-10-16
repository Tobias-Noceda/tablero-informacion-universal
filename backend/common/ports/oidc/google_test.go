package oidc

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	gojwt "github.com/golang-jwt/jwt/v5"
)

const (
	clientID     = "tiu-web"
	clientSecret = "client-secret"
	redirectURI  = "http://localhost/auth/google/callback"
	verifier     = "the-pkce-verifier"
	nonce        = "the-nonce"
	code         = "the-code"
)

// fakeIssuer is the smallest OpenID provider go-oidc accepts: discovery, a
// JWKS with the keys it publishes and a token endpoint that mints id_tokens.
type fakeIssuer struct {
	t      *testing.T
	server *httptest.Server

	mu            sync.Mutex
	published     map[string]*rsa.PrivateKey
	signingKid    string
	signingKey    *rsa.PrivateKey
	claims        gojwt.MapClaims
	discoveryDown bool
	tokenRequests []url.Values
}

func newFakeIssuer(t *testing.T) *fakeIssuer {
	t.Helper()
	f := &fakeIssuer{t: t, published: map[string]*rsa.PrivateKey{}}
	f.rotate("k1")

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", f.discovery)
	mux.HandleFunc("/jwks", f.jwks)
	mux.HandleFunc("/token", f.token)
	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)

	f.claims = gojwt.MapClaims{
		"iss":            f.server.URL,
		"aud":            clientID,
		"sub":            "google-sub-1",
		"email":          "Ana@Example.com",
		"email_verified": true,
		"name":           "Ana",
		"picture":        "https://lh3.example/ana.png",
		"nonce":          nonce,
		"iat":            time.Now().Unix(),
		"exp":            time.Now().Add(5 * time.Minute).Unix(),
	}
	return f
}

func (f *fakeIssuer) rotate(kid string) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		f.t.Fatal(err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.published[kid] = key
	f.signingKid, f.signingKey = kid, key
}

func (f *fakeIssuer) set(claim string, value any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.claims[claim] = value
}

func (f *fakeIssuer) discovery(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	down := f.discoveryDown
	f.mu.Unlock()
	if down {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, map[string]any{
		"issuer":                                f.server.URL,
		"authorization_endpoint":                f.server.URL + "/authorize",
		"token_endpoint":                        f.server.URL + "/token",
		"jwks_uri":                              f.server.URL + "/jwks",
		"id_token_signing_alg_values_supported": []string{"RS256"},
	})
}

func (f *fakeIssuer) jwks(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var keys []map[string]string
	for kid, key := range f.published {
		keys = append(keys, map[string]string{
			"kty": "RSA",
			"alg": "RS256",
			"use": "sig",
			"kid": kid,
			"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		})
	}
	writeJSON(w, map[string]any{"keys": keys})
}

func (f *fakeIssuer) token(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tokenRequests = append(f.tokenRequests, r.PostForm)

	id, secret, ok := r.BasicAuth()
	if !ok {
		id, secret = r.PostForm.Get("client_id"), r.PostForm.Get("client_secret")
	}
	if id != clientID || secret != clientSecret {
		w.WriteHeader(http.StatusUnauthorized)
		writeJSON(w, map[string]string{"error": "invalid_client"})
		return
	}
	if r.PostForm.Get("code") != code || r.PostForm.Get("code_verifier") != verifier || r.PostForm.Get("redirect_uri") != redirectURI {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": "invalid_grant"})
		return
	}

	token := gojwt.NewWithClaims(gojwt.SigningMethodRS256, f.claims)
	token.Header["kid"] = f.signingKid
	idToken, err := token.SignedString(f.signingKey)
	if err != nil {
		f.t.Fatal(err)
	}
	writeJSON(w, map[string]any{"access_token": "google-access", "token_type": "Bearer", "expires_in": 3600, "id_token": idToken})
}

func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}

func newGoogle(t *testing.T, f *fakeIssuer) *Google {
	t.Helper()
	return New(Config{Issuer: f.server.URL, ClientID: clientID, ClientSecret: clientSecret})
}

func TestAuthURL_PointsAtTheIssuerWithPKCEAndNonce(t *testing.T) {
	f := newFakeIssuer(t)
	google := newGoogle(t, f)

	raw, err := google.AuthURL("the-state", nonce, "the-challenge", redirectURI)
	if err != nil {
		t.Fatalf("auth url: %v", err)
	}

	parsed, _ := url.Parse(raw)
	if got := parsed.Scheme + "://" + parsed.Host + parsed.Path; got != f.server.URL+"/authorize" {
		t.Errorf("endpoint = %s", got)
	}
	query := parsed.Query()
	want := map[string]string{
		"response_type":         "code",
		"client_id":             clientID,
		"redirect_uri":          redirectURI,
		"state":                 "the-state",
		"nonce":                 nonce,
		"code_challenge":        "the-challenge",
		"code_challenge_method": "S256",
		"scope":                 "openid email profile",
	}
	for param, value := range want {
		if query.Get(param) != value {
			t.Errorf("%s = %q, want %q", param, query.Get(param), value)
		}
	}
	if strings.Contains(raw, clientSecret) {
		t.Error("the authorization URL must not carry the client secret")
	}
}

func TestComplete_ReturnsTheVerifiedIdentity(t *testing.T) {
	f := newFakeIssuer(t)
	google := newGoogle(t, f)

	identity, err := google.Complete(code, redirectURI, verifier, nonce)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}

	want := infrastructure.ExternalIdentity{
		Provider:      models.IdentityGoogle,
		Subject:       "google-sub-1",
		Email:         "Ana@Example.com",
		EmailVerified: true,
		Name:          "Ana",
		Picture:       "https://lh3.example/ana.png",
	}
	if *identity != want {
		t.Errorf("identity = %+v, want %+v", *identity, want)
	}
}

func TestComplete_ReportsAnUnverifiedEmail(t *testing.T) {
	f := newFakeIssuer(t)
	f.set("email_verified", false)

	identity, err := newGoogle(t, f).Complete(code, redirectURI, verifier, nonce)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if identity.EmailVerified {
		t.Error("email_verified:false must reach the caller")
	}
}

func TestComplete_RejectsTokensThatAreNotForThisSignIn(t *testing.T) {
	cases := []struct {
		name  string
		claim string
		value any
	}{
		{"another nonce", "nonce", "someone-elses-nonce"},
		{"another audience", "aud", "another-client"},
		{"another issuer", "iss", "https://evil.example"},
		{"expired", "exp", time.Now().Add(-time.Minute).Unix()},
		{"no subject", "sub", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeIssuer(t)
			f.set(tc.claim, tc.value)

			if _, err := newGoogle(t, f).Complete(code, redirectURI, verifier, nonce); !errors.Is(err, infrastructure.ErrIdentityProvider) {
				t.Errorf("err = %v, want ErrIdentityProvider", err)
			}
		})
	}
}

func TestComplete_SendsTheVerifierAndFailsWhenTheExchangeIsRefused(t *testing.T) {
	f := newFakeIssuer(t)
	google := newGoogle(t, f)

	if _, err := google.Complete(code, redirectURI, "a-guessed-verifier", nonce); !errors.Is(err, infrastructure.ErrIdentityProvider) {
		t.Errorf("err = %v, want ErrIdentityProvider", err)
	}
	if got := f.tokenRequests[0].Get("code_verifier"); got != "a-guessed-verifier" {
		t.Errorf("code_verifier = %q", got)
	}
	if f.tokenRequests[0].Get("grant_type") != "authorization_code" {
		t.Errorf("grant_type = %q", f.tokenRequests[0].Get("grant_type"))
	}
}

func TestComplete_FollowsAKeyRotation(t *testing.T) {
	f := newFakeIssuer(t)
	google := newGoogle(t, f)

	if _, err := google.Complete(code, redirectURI, verifier, nonce); err != nil {
		t.Fatalf("first sign-in: %v", err)
	}

	f.rotate("k2")
	if _, err := google.Complete(code, redirectURI, verifier, nonce); err != nil {
		t.Errorf("sign-in after rotation: %v", err)
	}
}

func TestComplete_RejectsASignatureFromAKeyTheIssuerNeverPublished(t *testing.T) {
	f := newFakeIssuer(t)
	google := newGoogle(t, f)

	forger, _ := rsa.GenerateKey(rand.Reader, 2048)
	f.mu.Lock()
	f.signingKey = forger
	f.mu.Unlock()

	if _, err := google.Complete(code, redirectURI, verifier, nonce); !errors.Is(err, infrastructure.ErrIdentityProvider) {
		t.Errorf("err = %v, want ErrIdentityProvider", err)
	}
}

func TestDiscovery_IsRetriedAfterAFailure(t *testing.T) {
	f := newFakeIssuer(t)
	f.discoveryDown = true
	google := newGoogle(t, f)

	if _, err := google.AuthURL("s", nonce, "c", redirectURI); !errors.Is(err, infrastructure.ErrIdentityProvider) {
		t.Fatalf("err = %v, want ErrIdentityProvider", err)
	}

	f.mu.Lock()
	f.discoveryDown = false
	f.mu.Unlock()

	if _, err := google.AuthURL("s", nonce, "c", redirectURI); err != nil {
		t.Errorf("discovery was not retried: %v", err)
	}
}
