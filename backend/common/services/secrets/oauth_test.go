package secrets

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/ports/crypto"
	"github.com/google/uuid"
)

func oauthService(t *testing.T, store *memoryStore, tokens *mocks.MockTokenClient, locks *mocks.MockLocker) *SecretsService {
	t.Helper()
	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x3D}, 32))
	sealer, err := crypto.NewFromKeys("1:" + key)
	if err != nil {
		t.Fatalf("sealer: %v", err)
	}
	boards := &mocks.MockDB{
		FindBoardFn: func(id uuid.UUID) (*models.Board, error) {
			return &models.Board{Id: id, Owner: owner}, nil
		},
	}
	if tokens == nil {
		tokens = &mocks.MockTokenClient{}
	}
	if locks == nil {
		locks = &mocks.MockLocker{}
	}
	return New(store, boards, sealer, tokens, locks, &mocks.MockHandshakeStore{})
}

func clientCredentials() *models.OAuth2Material {
	return &models.OAuth2Material{
		Flow:         models.OAuth2ClientCredentials,
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		TokenURL:     "https://provider.example/oauth/token",
		Scopes:       "read",
	}
}

func TestPutOAuth2_SealsTheClientSecret(t *testing.T) {
	store := newStore()
	srv := oauthService(t, store, nil, nil)
	board := uuid.New()

	if err := srv.PutOAuth2(board, owner, "SPOTIFY", clientCredentials()); err != nil {
		t.Fatalf("put: %v", err)
	}

	if len(store.rows) != 1 {
		t.Fatalf("stored %d rows", len(store.rows))
	}
	row := store.rows[0]
	if row.Kind != models.SecretOAuth2 {
		t.Errorf("kind = %q, want oauth2", row.Kind)
	}
	if bytes.Contains(row.Ciphertext, []byte("client-secret")) {
		t.Error("the client secret was stored in the clear")
	}
}

// A caller must not be able to plant a token it did not get from the provider.
func TestPutOAuth2_IgnoresCallerSuppliedTokens(t *testing.T) {
	store := newStore()
	srv := oauthService(t, store, nil, nil)
	board := uuid.New()

	material := clientCredentials()
	material.AccessToken = "planted"
	material.RefreshToken = "planted-refresh"
	material.ExpiresAt = time.Now().Add(time.Hour)

	if err := srv.PutOAuth2(board, owner, "SPOTIFY", material); err != nil {
		t.Fatalf("put: %v", err)
	}

	plaintext, err := srv.unseal(&store.rows[0])
	if err != nil {
		t.Fatalf("unseal: %v", err)
	}

	var stored models.OAuth2Material
	if err := json.Unmarshal(plaintext, &stored); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if stored.AccessToken != "" || stored.RefreshToken != "" {
		t.Errorf("caller tokens survived: %+v", stored)
	}
}

func TestPutOAuth2_Rejects(t *testing.T) {
	store := newStore()
	srv := oauthService(t, store, nil, nil)
	board := uuid.New()

	cases := map[string]func(*models.OAuth2Material){
		"unknown flow":     func(m *models.OAuth2Material) { m.Flow = "implicit" },
		"no client id":     func(m *models.OAuth2Material) { m.ClientID = "" },
		"no client secret": func(m *models.OAuth2Material) { m.ClientSecret = "" },
		"file scheme":      func(m *models.OAuth2Material) { m.TokenURL = "file:///etc/passwd" },
		"relative url":     func(m *models.OAuth2Material) { m.TokenURL = "/oauth/token" },
		"userinfo in url":  func(m *models.OAuth2Material) { m.TokenURL = "https://u:p@provider.example/t" },
	}

	for label, mutate := range cases {
		material := clientCredentials()
		mutate(material)
		if err := srv.PutOAuth2(board, owner, "SPOTIFY", material); err == nil {
			t.Errorf("%s: expected rejection", label)
		}
	}

	if len(store.rows) != 0 {
		t.Errorf("a rejected credential was persisted: %+v", store.rows)
	}
}

func TestPutOAuth2_OwnerOnly(t *testing.T) {
	store := newStore()
	srv := oauthService(t, store, nil, nil)

	if err := srv.PutOAuth2(uuid.New(), "stranger", "SPOTIFY", clientCredentials()); err == nil {
		t.Error("a stranger configured an OAuth2 credential")
	}
}

// From a post-it's point of view an OAuth2 credential is just a header value.
func TestResolve_OAuth2FetchesAndReturnsBearer(t *testing.T) {
	store := newStore()
	fetches := 0
	tokens := &mocks.MockTokenClient{
		FetchFn: func(m *models.OAuth2Material) error {
			fetches++
			m.AccessToken = "at-12345"
			m.TokenType = "Bearer"
			m.ExpiresAt = time.Now().Add(time.Hour)
			return nil
		},
	}
	srv := oauthService(t, store, tokens, nil)
	board := uuid.New()

	if err := srv.PutOAuth2(board, owner, "SPOTIFY", clientCredentials()); err != nil {
		t.Fatalf("put: %v", err)
	}

	resolved, err := srv.Resolve(board, []string{"SPOTIFY"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved["$SPOTIFY"] != "Bearer at-12345" {
		t.Errorf("got %q, want the bearer header", resolved["$SPOTIFY"])
	}
	if fetches != 1 {
		t.Errorf("fetched %d times, want 1", fetches)
	}
}

// The token just fetched must be persisted, or every execution pays for a new one.
func TestResolve_OAuth2CachesTheToken(t *testing.T) {
	store := newStore()
	fetches := 0
	tokens := &mocks.MockTokenClient{
		FetchFn: func(m *models.OAuth2Material) error {
			fetches++
			m.AccessToken = "at-12345"
			m.ExpiresAt = time.Now().Add(time.Hour)
			return nil
		},
	}
	srv := oauthService(t, store, tokens, nil)
	board := uuid.New()

	_ = srv.PutOAuth2(board, owner, "SPOTIFY", clientCredentials())

	for range 3 {
		if _, err := srv.Resolve(board, []string{"SPOTIFY"}); err != nil {
			t.Fatalf("resolve: %v", err)
		}
	}

	if fetches != 1 {
		t.Errorf("hit the token endpoint %d times, want 1", fetches)
	}
}

// A token that expires inside the margin is renewed before it is handed out.
func TestResolve_OAuth2RenewsWithinMargin(t *testing.T) {
	store := newStore()
	fetches := 0
	tokens := &mocks.MockTokenClient{
		FetchFn: func(m *models.OAuth2Material) error {
			fetches++
			m.AccessToken = "fresh"
			// Always lands inside the refresh margin.
			m.ExpiresAt = time.Now().Add(models.REFRESH_MARGIN / 2)
			return nil
		},
	}
	srv := oauthService(t, store, tokens, nil)
	board := uuid.New()

	_ = srv.PutOAuth2(board, owner, "SPOTIFY", clientCredentials())

	for range 2 {
		if _, err := srv.Resolve(board, []string{"SPOTIFY"}); err != nil {
			t.Fatalf("resolve: %v", err)
		}
	}

	if fetches != 2 {
		t.Errorf("fetched %d times, want a renewal on each resolve", fetches)
	}
}

// Providers that rotate refresh tokens invalidate the old one immediately, so
// a replacement handed back by Fetch has to survive to the next refresh.
func TestResolve_OAuth2PersistsRotatedRefreshToken(t *testing.T) {
	store := newStore()
	tokens := &mocks.MockTokenClient{
		FetchFn: func(m *models.OAuth2Material) error {
			m.AccessToken = "at"
			m.RefreshToken = "rotated-refresh"
			m.ExpiresAt = time.Now().Add(time.Hour)
			return nil
		},
	}
	srv := oauthService(t, store, tokens, nil)
	board := uuid.New()

	_ = srv.PutOAuth2(board, owner, "SPOTIFY", clientCredentials())
	if _, err := srv.Resolve(board, []string{"SPOTIFY"}); err != nil {
		t.Fatalf("resolve: %v", err)
	}

	latest := store.rows[len(store.rows)-1]
	plaintext, err := srv.unseal(&latest)
	if err != nil {
		t.Fatalf("unseal: %v", err)
	}

	var stored models.OAuth2Material
	_ = json.Unmarshal(plaintext, &stored)
	if stored.RefreshToken != "rotated-refresh" {
		t.Errorf("refresh token = %q, want the rotated one to be stored", stored.RefreshToken)
	}
}

// Losing the lock means another worker is already refreshing; this one must
// not also spend the refresh token.
func TestResolve_OAuth2DoesNotRefreshWithoutTheLock(t *testing.T) {
	store := newStore()
	fetched := false
	tokens := &mocks.MockTokenClient{
		FetchFn: func(m *models.OAuth2Material) error {
			fetched = true
			return nil
		},
	}
	locks := &mocks.MockLocker{
		AcquireFn: func(string, time.Duration) (string, bool, error) { return "", false, nil },
	}
	srv := oauthService(t, store, tokens, locks)
	board := uuid.New()

	_ = srv.PutOAuth2(board, owner, "SPOTIFY", clientCredentials())

	_, err := srv.Resolve(board, []string{"SPOTIFY"})
	if err == nil {
		t.Fatal("expected the loser of the race to report a retryable error")
	}
	if !strings.Contains(err.Error(), "in progress") {
		t.Errorf("error = %v, want it to name the concurrent refresh", err)
	}
	if fetched {
		t.Error("refreshed without holding the lock")
	}
}

// The winner writes a usable token; the loser re-reads it instead of failing.
func TestResolve_OAuth2LoserPicksUpTheWinnersToken(t *testing.T) {
	store := newStore()
	board := uuid.New()

	// Seed a credential that still needs a token, then let a winner refresh it
	// into a second, usable revision.
	seeder := oauthService(t, store, nil, nil)
	if err := seeder.PutOAuth2(board, owner, "SPOTIFY", clientCredentials()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	stale := store.rows[0]

	winner := oauthService(t, store, &mocks.MockTokenClient{
		FetchFn: func(m *models.OAuth2Material) error {
			m.AccessToken = "winners-token"
			m.ExpiresAt = time.Now().Add(time.Hour)
			return nil
		},
	}, nil)
	if _, err := winner.Resolve(board, []string{"SPOTIFY"}); err != nil {
		t.Fatalf("winner resolve: %v", err)
	}
	refreshed := store.rows[0]

	// The loser's first read is the stale revision, so it tries to refresh;
	// it loses the lock, re-reads, and finds what the winner stored.
	reads := 0
	store.FindSecretsFn = func(_ uuid.UUID, _ []string) ([]models.Secret, error) {
		reads++
		if reads == 1 {
			return []models.Secret{stale}, nil
		}
		return []models.Secret{refreshed}, nil
	}

	loser := oauthService(t, store, &mocks.MockTokenClient{
		FetchFn: func(*models.OAuth2Material) error {
			t.Error("the loser refreshed despite not holding the lock")
			return nil
		},
	}, &mocks.MockLocker{
		AcquireFn: func(string, time.Duration) (string, bool, error) { return "", false, nil },
	})

	resolved, err := loser.Resolve(board, []string{"SPOTIFY"})
	if err != nil {
		t.Fatalf("loser resolve: %v", err)
	}
	if !strings.Contains(resolved["$SPOTIFY"], "winners-token") {
		t.Errorf("loser got %q, want the winner's token", resolved["$SPOTIFY"])
	}
	if reads != 2 {
		t.Errorf("store read %d times, want a re-read after losing the lock", reads)
	}
}
