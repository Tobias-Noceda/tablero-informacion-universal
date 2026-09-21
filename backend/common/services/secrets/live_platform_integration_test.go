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

// The same handshake as TestLiveAuthorizationCode, but the application
// credential belongs to the platform and the grant only ever holds tokens.
func TestLivePlatformConnect(t *testing.T) {
	original := safehttp.IsSafeIP
	safehttp.IsSafeIP = func(net.IP) bool { return true }
	t.Cleanup(func() { safehttp.IsSafeIP = original })

	const mock models.OAuthProvider = "mock"
	models.OAuthProviders[mock] = models.OAuthProviderConfig{
		AuthURL:    mockIssuer + "/authorize",
		TokenURL:   mockIssuer + "/token",
		Scopes:     "openid offline_access",
		Credential: "MOCK_OAUTH_CLIENT",
	}
	t.Cleanup(func() { delete(models.OAuthProviders, mock) })

	store := newStore()
	srv := handshakeService(t, store, &mocks.MockHandshakeStore{}, nil)
	srv.tokens = oauth.New()

	if err := srv.PutOAuth2Client(anonymous, mock, "test-client", "test-secret"); err != nil {
		t.Fatalf("provision client: %v", err)
	}

	board := models.BoardScope(uuid.New())
	redirect := "http://localhost:9999/cb"

	target, err := srv.Connect(board, owner, mock, "MOCKGRANT", redirect)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	parsed, _ := url.Parse(target)
	state := parsed.Query().Get("state")
	if parsed.Query().Get("client_id") != "test-client" {
		t.Fatalf("client_id = %q, want the platform's", parsed.Query().Get("client_id"))
	}
	t.Logf("connect   : state=%s...", state[:16])

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

	location, _ := url.Parse(res.Header.Get("Location"))
	code := location.Query().Get("code")
	if code == "" {
		t.Fatalf("provider returned no code (status %d)", res.StatusCode)
	}

	if err := srv.Callback(state, code); err != nil {
		t.Fatalf("callback: %v", err)
	}

	grant := func() models.OAuth2Material {
		for _, row := range store.rows {
			if row.Scope == board {
				plaintext, err := srv.unseal(&row)
				if err != nil {
					t.Fatalf("unseal: %v", err)
				}
				var m models.OAuth2Material
				_ = json.Unmarshal(plaintext, &m)
				return m
			}
		}
		t.Fatal("no grant stored")
		return models.OAuth2Material{}
	}

	stored := grant()
	if stored.RefreshToken == "" || stored.AccessToken == "" {
		t.Fatal("no tokens after the exchange")
	}
	if stored.ClientSecret != "" || stored.ClientID != "" || stored.TokenURL != "" {
		t.Fatalf("platform credentials persisted into the grant: %+v", stored)
	}
	t.Logf("exchange  : tokens stored, client fields empty on disk")

	firstAccess := stored.AccessToken
	stored.ExpiresAt = time.Now().Add(-time.Minute)
	if err := srv.sealMaterial(board, "MOCKGRANT", &stored, true); err != nil {
		t.Fatalf("reseal expired: %v", err)
	}

	resolved, err := srv.Resolve(board, []string{"MOCKGRANT"})
	if err != nil {
		t.Fatalf("resolve after expiry: %v", err)
	}
	if !strings.HasPrefix(resolved["$MOCKGRANT"], "Bearer ") {
		t.Fatalf("resolved = %q", resolved["$MOCKGRANT"])
	}

	refreshed := grant()
	if refreshed.AccessToken == firstAccess {
		t.Error("the access token was not renewed")
	}
	if refreshed.ClientSecret != "" {
		t.Error("the refresh persisted the platform client secret")
	}
	t.Logf("refresh   : renewed with the platform client, expires in %v", time.Until(refreshed.ExpiresAt).Round(time.Second))
}
