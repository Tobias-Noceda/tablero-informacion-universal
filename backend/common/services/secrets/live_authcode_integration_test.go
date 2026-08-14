//go:build integration

package secrets

import (
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/ports/oauth"
	"github.com/Secreto31126/tesis/common/ports/safehttp"
	"github.com/google/uuid"
)

const mockIssuer = "http://localhost:8899/default"

func TestLiveAuthorizationCode(t *testing.T) {
	// The provider is on loopback, which production rightly blocks.
	original := safehttp.IsSafeIP
	safehttp.IsSafeIP = func(net.IP) bool { return true }
	t.Cleanup(func() { safehttp.IsSafeIP = original })

	store := newStore()
	handshakes := &mocks.MockHandshakeStore{}
	srv := handshakeService(t, store, handshakes, nil)
	srv.tokens = oauth.New()

	board := uuid.New()
	redirect := "http://localhost:9999/cb"

	material := &models.OAuth2Material{
		Flow:         models.OAuth2AuthorizationCode,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		TokenURL:     mockIssuer + "/token",
		AuthURL:      mockIssuer + "/authorize",
		Scopes:       "openid offline_access",
	}

	if err := srv.PutOAuth2(board, owner, "MOCKPROVIDER", material); err != nil {
		t.Fatalf("configure: %v", err)
	}

	target, err := srv.Authorize(board, owner, "MOCKPROVIDER", redirect)
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	parsed, _ := url.Parse(target)
	state := parsed.Query().Get("state")
	t.Logf("authorize : challenge=%s... state=%s...",
		parsed.Query().Get("code_challenge")[:16], state[:16])

	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	res, err := client.PostForm(target, url.Values{"username": {"alice"}})
	if err != nil {
		t.Fatalf("consent: %v", err)
	}
	defer res.Body.Close()

	location, err := url.Parse(res.Header.Get("Location"))
	if err != nil {
		t.Fatalf("redirect: %v", err)
	}
	code := location.Query().Get("code")
	if code == "" {
		t.Fatalf("provider returned no code (status %d)", res.StatusCode)
	}
	if location.Query().Get("state") != state {
		t.Fatalf("provider echoed state %q, want %q", location.Query().Get("state"), state)
	}
	t.Logf("consent   : code=%s... state echoed back intact", code[:16])

	if err := srv.Callback(state, code); err != nil {
		t.Fatalf("callback: %v", err)
	}

	plaintext, err := srv.unseal(&store.rows[0])
	if err != nil {
		t.Fatalf("unseal: %v", err)
	}
	var stored models.OAuth2Material
	_ = json.Unmarshal(plaintext, &stored)

	if stored.AccessToken == "" {
		t.Fatal("no access token after the exchange")
	}
	if stored.RefreshToken == "" {
		t.Fatal("no refresh token after the exchange")
	}
	t.Logf("exchange  : access=%s... refresh=%s...", stored.AccessToken[:20], stored.RefreshToken[:20])

	if err := srv.Callback(state, code); err == nil {
		t.Error("the state was accepted a second time")
	}
	t.Log("replay    : the spent state was rejected")

	firstAccess := stored.AccessToken
	firstRefresh := stored.RefreshToken

	stored.ExpiresAt = time.Now().Add(-time.Minute)
	expired, _ := json.Marshal(&stored)
	if err := srv.seal(board, "MOCKPROVIDER", models.SecretOAuth2, expired); err != nil {
		t.Fatalf("reseal: %v", err)
	}

	resolved, err := srv.Resolve(board, []string{"MOCKPROVIDER"})
	if err != nil {
		t.Fatalf("resolve after expiry: %v", err)
	}
	if !strings.HasPrefix(resolved["$MOCKPROVIDER"], "Bearer ") {
		t.Fatalf("resolved = %q", resolved["$MOCKPROVIDER"])
	}

	plaintext, _ = srv.unseal(&store.rows[0])
	var refreshed models.OAuth2Material
	_ = json.Unmarshal(plaintext, &refreshed)

	if refreshed.AccessToken == firstAccess {
		t.Error("the access token was not renewed")
	}
	if refreshed.RefreshToken == "" {
		t.Error("the refresh token was lost during the refresh")
	}
	if time.Until(refreshed.ExpiresAt) <= 0 {
		t.Error("the renewed token is already expired")
	}

	rotated := refreshed.RefreshToken != firstRefresh
	t.Logf("refresh   : new access token, refresh rotated=%v, expires in %v",
		rotated, time.Until(refreshed.ExpiresAt).Round(time.Second))
}
