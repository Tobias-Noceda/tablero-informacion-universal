package secrets

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

const redirectURI = "https://tablero.example/oauth2/callback"

func authCode() *models.OAuth2Material {
	return &models.OAuth2Material{
		Flow:         models.OAuth2AuthorizationCode,
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		TokenURL:     "https://provider.example/oauth/token",
		AuthURL:      "https://provider.example/oauth/authorize",
		Scopes:       "playlists",
	}
}

// handshakeService keeps one shared handshake store so Authorize and Callback
// see the same pending entries.
func handshakeService(t *testing.T, store *memoryStore, handshakes *mocks.MockHandshakeStore, tokens *mocks.MockTokenClient) *SecretsService {
	t.Helper()
	srv := oauthService(t, store, tokens, nil)
	srv.handshakes = handshakes
	return srv
}

func TestAuthorize_BuildsProviderURL(t *testing.T) {
	store := newStore()
	handshakes := &mocks.MockHandshakeStore{}
	srv := handshakeService(t, store, handshakes, nil)
	board := uuid.New()

	if err := srv.PutOAuth2(board, owner, "SPOTIFY", authCode()); err != nil {
		t.Fatalf("put: %v", err)
	}

	target, err := srv.Authorize(board, owner, "SPOTIFY", redirectURI)
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}

	parsed, err := url.Parse(target)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	q := parsed.Query()

	if parsed.Host != "provider.example" || parsed.Path != "/oauth/authorize" {
		t.Errorf("target = %s, want the provider's authorize endpoint", target)
	}
	if q.Get("response_type") != "code" {
		t.Errorf("response_type = %q", q.Get("response_type"))
	}
	if q.Get("client_id") != "client-id" {
		t.Errorf("client_id = %q", q.Get("client_id"))
	}
	if q.Get("redirect_uri") != redirectURI {
		t.Errorf("redirect_uri = %q", q.Get("redirect_uri"))
	}
	if q.Get("scope") != "playlists" {
		t.Errorf("scope = %q", q.Get("scope"))
	}
	if q.Get("code_challenge_method") != "S256" {
		t.Errorf("challenge method = %q, want S256", q.Get("code_challenge_method"))
	}
	if q.Get("state") == "" || q.Get("code_challenge") == "" {
		t.Error("state and code_challenge must both be present")
	}

	// The client secret must never travel on the front channel.
	if strings.Contains(target, "client-secret") {
		t.Errorf("the authorization URL leaked the client secret: %s", target)
	}
}

// The challenge sent to the provider must be the hash of the verifier that is
// kept server-side, or PKCE proves nothing.
func TestAuthorize_ChallengeMatchesStoredVerifier(t *testing.T) {
	store := newStore()
	handshakes := &mocks.MockHandshakeStore{}
	srv := handshakeService(t, store, handshakes, nil)
	board := uuid.New()

	_ = srv.PutOAuth2(board, owner, "SPOTIFY", authCode())
	target, err := srv.Authorize(board, owner, "SPOTIFY", redirectURI)
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}

	parsed, _ := url.Parse(target)
	state := parsed.Query().Get("state")
	challenge := parsed.Query().Get("code_challenge")

	raw, err := handshakes.Take(handshakeKey(state))
	if err != nil {
		t.Fatalf("the handshake was not stored under its state: %v", err)
	}

	var stored pending
	if err := json.Unmarshal(raw, &stored); err != nil {
		t.Fatalf("decode: %v", err)
	}

	sum := sha256.Sum256([]byte(stored.Verifier))
	if base64.RawURLEncoding.EncodeToString(sum[:]) != challenge {
		t.Error("the challenge is not S256 of the stored verifier")
	}
	if strings.Contains(target, stored.Verifier) {
		t.Error("the verifier itself was sent to the provider")
	}
}

func TestAuthorize_Rejects(t *testing.T) {
	store := newStore()
	srv := handshakeService(t, store, &mocks.MockHandshakeStore{}, nil)
	board := uuid.New()

	// A client_credentials credential has nothing to consent to.
	_ = srv.PutOAuth2(board, owner, "MACHINE", clientCredentials())
	if _, err := srv.Authorize(board, owner, "MACHINE", redirectURI); err == nil {
		t.Error("client_credentials should not start a consent handshake")
	}

	_ = srv.PutOAuth2(board, owner, "SPOTIFY", authCode())

	if _, err := srv.Authorize(board, "stranger", "SPOTIFY", redirectURI); err == nil {
		t.Error("a stranger started a handshake")
	}
	if _, err := srv.Authorize(board, owner, "SPOTIFY", "javascript:alert(1)"); err == nil {
		t.Error("a non-http redirect_uri was accepted")
	}
	if _, err := srv.Authorize(board, owner, "MISSING", redirectURI); err == nil {
		t.Error("a handshake started for a credential that does not exist")
	}
}

func TestCallback_ExchangesAndStoresTokens(t *testing.T) {
	store := newStore()
	handshakes := &mocks.MockHandshakeStore{}

	var gotCode, gotRedirect, gotVerifier string
	tokens := &mocks.MockTokenClient{
		ExchangeFn: func(m *models.OAuth2Material, code, redirect, verifier string) error {
			gotCode, gotRedirect, gotVerifier = code, redirect, verifier
			m.AccessToken = "at"
			m.RefreshToken = "rt"
			m.ExpiresAt = time.Now().Add(time.Hour)
			return nil
		},
	}

	srv := handshakeService(t, store, handshakes, tokens)
	board := uuid.New()
	_ = srv.PutOAuth2(board, owner, "SPOTIFY", authCode())

	target, err := srv.Authorize(board, owner, "SPOTIFY", redirectURI)
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	parsed, _ := url.Parse(target)
	state := parsed.Query().Get("state")

	if err := srv.Callback(state, "the-code"); err != nil {
		t.Fatalf("callback: %v", err)
	}

	if gotCode != "the-code" || gotRedirect != redirectURI || gotVerifier == "" {
		t.Errorf("exchange got code=%q redirect=%q verifier=%q", gotCode, gotRedirect, gotVerifier)
	}

	plaintext, err := srv.unseal(&store.rows[0])
	if err != nil {
		t.Fatalf("unseal: %v", err)
	}
	var stored models.OAuth2Material
	_ = json.Unmarshal(plaintext, &stored)
	if stored.RefreshToken != "rt" {
		t.Errorf("refresh token = %q, want it persisted", stored.RefreshToken)
	}
}

// A state is spent on first use, so an intercepted callback cannot be replayed.
func TestCallback_StateIsSingleUse(t *testing.T) {
	store := newStore()
	handshakes := &mocks.MockHandshakeStore{}
	exchanges := 0
	tokens := &mocks.MockTokenClient{
		ExchangeFn: func(m *models.OAuth2Material, _, _, _ string) error {
			exchanges++
			m.AccessToken = "at"
			m.RefreshToken = "rt"
			return nil
		},
	}

	srv := handshakeService(t, store, handshakes, tokens)
	board := uuid.New()
	_ = srv.PutOAuth2(board, owner, "SPOTIFY", authCode())

	target, _ := srv.Authorize(board, owner, "SPOTIFY", redirectURI)
	parsed, _ := url.Parse(target)
	state := parsed.Query().Get("state")

	if err := srv.Callback(state, "the-code"); err != nil {
		t.Fatalf("first callback: %v", err)
	}
	if err := srv.Callback(state, "the-code"); err == nil {
		t.Error("the same state was accepted twice")
	}
	if exchanges != 1 {
		t.Errorf("exchanged %d times, want 1", exchanges)
	}
}

// Without a matching state an attacker cannot get their own code redeemed
// against someone else's credential.
func TestCallback_RejectsUnknownState(t *testing.T) {
	store := newStore()
	exchanged := false
	tokens := &mocks.MockTokenClient{
		ExchangeFn: func(*models.OAuth2Material, string, string, string) error {
			exchanged = true
			return nil
		},
	}
	srv := handshakeService(t, store, &mocks.MockHandshakeStore{}, tokens)

	for _, c := range []struct{ state, code string }{
		{"forged-state", "attacker-code"},
		{"", "attacker-code"},
		{"forged-state", ""},
	} {
		if err := srv.Callback(c.state, c.code); err == nil {
			t.Errorf("state=%q code=%q was accepted", c.state, c.code)
		}
	}

	if exchanged {
		t.Error("an exchange happened without a valid handshake")
	}
}
