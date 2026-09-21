package secrets

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

const (
	platformClientID     = "platform-client-id"
	platformClientSecret = "platform-client-secret"
)

func googleConfig() models.OAuthProviderConfig {
	return models.OAuthProviders[models.ProviderGoogle]
}

func withGoogleClient(t *testing.T, srv *SecretsService) {
	t.Helper()
	if err := srv.PutOAuth2Client(anonymous, models.ProviderGoogle, platformClientID, platformClientSecret); err != nil {
		t.Fatalf("put client: %v", err)
	}
}

func storedMaterial(t *testing.T, srv *SecretsService, store *memoryStore, scope models.SecretScope, name string) models.OAuth2Material {
	t.Helper()
	for _, row := range store.rows {
		if row.Scope == scope && row.Name == name {
			plaintext, err := srv.unseal(&row)
			if err != nil {
				t.Fatalf("unseal: %v", err)
			}
			var material models.OAuth2Material
			_ = json.Unmarshal(plaintext, &material)
			return material
		}
	}
	t.Fatalf("no %s/%s stored", scope.Key(), name)
	return models.OAuth2Material{}
}

func TestPutOAuth2Client_StoresUnderTheProviderNameInTheSystemScope(t *testing.T) {
	store := newStore()
	srv := service(t, store)

	withGoogleClient(t, srv)

	if len(store.rows) != 1 {
		t.Fatalf("stored %d rows", len(store.rows))
	}
	row := store.rows[0]
	if row.Scope != models.SystemScope || row.Name != string(googleConfig().Credential) || row.Kind != models.SecretOAuth2Client {
		t.Errorf("row = scope %s name %s kind %s", row.Scope.Key(), row.Name, row.Kind)
	}
	if strings.Contains(string(row.Ciphertext), platformClientSecret) {
		t.Error("client secret stored in the clear")
	}
}

func TestPutOAuth2Client_Rejects(t *testing.T) {
	srv := service(t, newStore())

	if err := srv.PutOAuth2Client(anonymous, "myspace", "id", "secret"); err == nil {
		t.Error("an unknown provider was accepted")
	}
	if err := srv.PutOAuth2Client(anonymous, models.ProviderGoogle, "", "secret"); err == nil {
		t.Error("an empty client id was accepted")
	}
	if err := srv.PutOAuth2Client(anonymous, models.ProviderGoogle, "id", ""); err == nil {
		t.Error("an empty client secret was accepted")
	}
}

// A plain Put must not be able to plant a client credential; only the
// provider-aware path writes that kind.
func TestPut_RejectsTheClientKind(t *testing.T) {
	srv := service(t, newStore())

	if err := srv.Put(models.SystemScope, anonymous, "X", models.SecretOAuth2Client, `{"client_id":"a","client_secret":"b"}`); err == nil {
		t.Error("oauth2_client was accepted through Put")
	}
}

func TestConnect_StoresAGrantWithoutPlatformCredentials(t *testing.T) {
	store := newStore()
	handshakes := &mocks.MockHandshakeStore{}
	srv := handshakeService(t, store, handshakes, nil)
	withGoogleClient(t, srv)
	board := models.BoardScope(uuid.New())

	target, err := srv.Connect(board, owner, models.ProviderGoogle, "GOOGLE", redirectURI)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	parsed, _ := url.Parse(target)
	authURL, _ := url.Parse(googleConfig().AuthURL)
	if parsed.Host != authURL.Host || parsed.Path != authURL.Path {
		t.Errorf("target = %s, want the provider's authorize endpoint", target)
	}
	if parsed.Query().Get("access_type") != "offline" {
		t.Error("the provider's own query parameters were dropped")
	}
	if parsed.Query().Get("client_id") != platformClientID {
		t.Errorf("client_id = %q, want the platform's", parsed.Query().Get("client_id"))
	}
	if parsed.Query().Get("scope") != googleConfig().Scopes {
		t.Errorf("scope = %q, want the provider's", parsed.Query().Get("scope"))
	}
	if strings.Contains(target, platformClientSecret) {
		t.Error("the client secret went to the front channel")
	}

	grant := storedMaterial(t, srv, store, board, "GOOGLE")
	if grant.Provider != models.ProviderGoogle || grant.Flow != models.OAuth2AuthorizationCode {
		t.Errorf("grant = %+v", grant)
	}
	if grant.ClientID != "" || grant.ClientSecret != "" || grant.TokenURL != "" || grant.AuthURL != "" || grant.Scopes != "" {
		t.Errorf("platform credentials were copied into the grant: %+v", grant)
	}

	metas, _ := srv.List(board, owner)
	if len(metas) != 1 || metas[0].Provider != models.ProviderGoogle || metas[0].Authorized {
		t.Errorf("metas = %+v, want an unauthorized google grant", metas)
	}
}

func TestConnect_Rejects(t *testing.T) {
	store := newStore()
	srv := handshakeService(t, store, &mocks.MockHandshakeStore{}, nil)
	withGoogleClient(t, srv)
	board := models.BoardScope(uuid.New())

	if _, err := srv.Connect(board, owner, "myspace", "X", redirectURI); err == nil {
		t.Error("an unknown provider was accepted")
	}
	if _, err := srv.Connect(board, stranger, models.ProviderGoogle, "GOOGLE", redirectURI); !errors.Is(err, ErrForbidden) {
		t.Errorf("a stranger connected: %v", err)
	}
	if _, err := srv.Connect(models.SystemScope, anonymous, models.ProviderGoogle, "GOOGLE", redirectURI); err == nil {
		t.Error("the system scope consented to something")
	}
	if _, err := srv.Connect(board, owner, models.ProviderGoogle, "lower", redirectURI); err == nil {
		t.Error("a bad name was accepted")
	}

	if len(store.rows) != 1 {
		t.Errorf("a rejected connect left rows behind: %d", len(store.rows))
	}
}

func TestConnect_FailsClearlyWhenTheProviderIsNotProvisioned(t *testing.T) {
	store := newStore()
	srv := handshakeService(t, store, &mocks.MockHandshakeStore{}, nil)
	board := models.BoardScope(uuid.New())

	_, err := srv.Connect(board, owner, models.ProviderGoogle, "GOOGLE", redirectURI)
	if !errors.Is(err, ErrProviderNotConfigured) {
		t.Fatalf("err = %v, want ErrProviderNotConfigured", err)
	}
	if len(store.rows) != 0 {
		t.Error("a grant was stored for a provider that cannot be used")
	}
}

func TestCallback_ExchangesWithThePlatformClient(t *testing.T) {
	store := newStore()
	handshakes := &mocks.MockHandshakeStore{}

	var seen models.OAuth2Material
	tokens := &mocks.MockTokenClient{
		ExchangeFn: func(m *models.OAuth2Material, _, _, _ string) error {
			seen = *m
			m.AccessToken = "at"
			m.RefreshToken = "rt"
			m.ExpiresAt = time.Now().Add(time.Hour)
			return nil
		},
	}
	srv := handshakeService(t, store, handshakes, tokens)
	withGoogleClient(t, srv)
	board := models.BoardScope(uuid.New())

	target, _ := srv.Connect(board, owner, models.ProviderGoogle, "GOOGLE", redirectURI)
	parsed, _ := url.Parse(target)

	if err := srv.Callback(parsed.Query().Get("state"), "the-code"); err != nil {
		t.Fatalf("callback: %v", err)
	}

	if seen.ClientID != platformClientID || seen.ClientSecret != platformClientSecret || seen.TokenURL != googleConfig().TokenURL {
		t.Errorf("exchange used %+v, want the platform client and the provider's token url", seen)
	}

	grant := storedMaterial(t, srv, store, board, "GOOGLE")
	if grant.RefreshToken != "rt" {
		t.Errorf("refresh token = %q, want it persisted", grant.RefreshToken)
	}
	if grant.ClientSecret != "" || grant.ClientID != "" {
		t.Errorf("platform credentials leaked into the stored grant: %+v", grant)
	}

	metas, _ := srv.List(board, owner)
	if !metas[0].Authorized {
		t.Error("the grant is not marked authorized")
	}
}

func TestResolve_RefreshesAProviderGrantWithThePlatformClient(t *testing.T) {
	store := newStore()
	handshakes := &mocks.MockHandshakeStore{}

	var seen models.OAuth2Material
	tokens := &mocks.MockTokenClient{
		ExchangeFn: func(m *models.OAuth2Material, _, _, _ string) error {
			m.AccessToken = "at"
			m.RefreshToken = "rt"
			m.ExpiresAt = time.Now().Add(-time.Minute)
			return nil
		},
		FetchFn: func(m *models.OAuth2Material) error {
			seen = *m
			m.AccessToken = "fresh"
			m.ExpiresAt = time.Now().Add(time.Hour)
			return nil
		},
	}
	srv := handshakeService(t, store, handshakes, tokens)
	withGoogleClient(t, srv)
	board := models.BoardScope(uuid.New())

	target, _ := srv.Connect(board, owner, models.ProviderGoogle, "GOOGLE", redirectURI)
	parsed, _ := url.Parse(target)
	_ = srv.Callback(parsed.Query().Get("state"), "the-code")

	resolved, err := srv.Resolve(board, []string{"GOOGLE"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved["$GOOGLE"] != "Bearer fresh" {
		t.Errorf("resolved = %v", resolved)
	}
	if seen.ClientSecret != platformClientSecret || seen.RefreshToken != "rt" {
		t.Errorf("refresh used %+v, want the platform client and the stored refresh token", seen)
	}

	grant := storedMaterial(t, srv, store, board, "GOOGLE")
	if grant.ClientSecret != "" {
		t.Error("the refresh persisted the platform client secret into the grant")
	}
}

// Rotating the platform client must not require every user to consent again:
// the grant only holds tokens, so the next refresh simply uses the new client.
func TestResolve_PicksUpARotatedPlatformClient(t *testing.T) {
	store := newStore()
	handshakes := &mocks.MockHandshakeStore{}
	var seenSecret string
	tokens := &mocks.MockTokenClient{
		ExchangeFn: func(m *models.OAuth2Material, _, _, _ string) error {
			m.AccessToken = "at"
			m.RefreshToken = "rt"
			m.ExpiresAt = time.Now().Add(-time.Minute)
			return nil
		},
		FetchFn: func(m *models.OAuth2Material) error {
			seenSecret = m.ClientSecret
			m.AccessToken = "fresh"
			m.ExpiresAt = time.Now().Add(time.Hour)
			return nil
		},
	}
	srv := handshakeService(t, store, handshakes, tokens)
	withGoogleClient(t, srv)
	board := models.BoardScope(uuid.New())

	target, _ := srv.Connect(board, owner, models.ProviderGoogle, "GOOGLE", redirectURI)
	parsed, _ := url.Parse(target)
	_ = srv.Callback(parsed.Query().Get("state"), "the-code")

	if err := srv.PutOAuth2Client(anonymous, models.ProviderGoogle, "new-id", "new-secret"); err != nil {
		t.Fatalf("rotate client: %v", err)
	}

	if _, err := srv.Resolve(board, []string{"GOOGLE"}); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if seenSecret != "new-secret" {
		t.Errorf("refresh used %q, want the rotated client", seenSecret)
	}
}

// The system scope is the platform itself: it can hold machine credentials,
// but never a consent-based grant.
func TestPutOAuth2_SystemScopeOnlyTakesClientCredentials(t *testing.T) {
	store := newStore()
	srv := service(t, store)

	if err := srv.PutOAuth2(models.SystemScope, anonymous, "MACHINE", clientCredentials()); err != nil {
		t.Errorf("client_credentials rejected in the system scope: %v", err)
	}
	if err := srv.PutOAuth2(models.SystemScope, anonymous, "CONSENT", authCode()); err == nil {
		t.Error("authorization_code accepted in the system scope")
	}
}

func TestProviders_ReportsWhichAreProvisioned(t *testing.T) {
	store := newStore()
	srv := service(t, store)
	withGoogleClient(t, srv)

	statuses, err := srv.Providers()
	if err != nil {
		t.Fatalf("providers: %v", err)
	}

	byName := map[models.OAuthProvider]bool{}
	for _, s := range statuses {
		byName[s.Provider] = s.Configured
	}
	if configured, listed := byName[models.ProviderGoogle]; !listed || !configured {
		t.Errorf("google = listed %v configured %v", listed, configured)
	}
	if configured, listed := byName[models.ProviderDiscord]; !listed || configured {
		t.Errorf("discord = listed %v configured %v, want listed and unconfigured", listed, configured)
	}
}
