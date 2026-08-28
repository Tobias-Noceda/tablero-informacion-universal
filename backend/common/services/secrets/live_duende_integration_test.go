//go:build integration

package secrets

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/ports/oauth"
	"github.com/google/uuid"
)

func TestLiveDuende(t *testing.T) {
	store := newStore()
	srv := oauthService(t, store, nil, nil)
	// Swap the double for the real token client.
	srv.tokens = oauth.New()

	board := uuid.New()

	material := &models.OAuth2Material{
		Flow:         models.OAuth2ClientCredentials,
		ClientID:     "m2m",
		ClientSecret: "secret",
		TokenURL:     "https://demo.duendesoftware.com/connect/token",
		Scopes:       "api",
	}

	if err := srv.PutOAuth2(board, owner, "DUENDE", material); err != nil {
		t.Fatalf("configure: %v", err)
	}

	row := store.rows[0]
	if strings.Contains(string(row.Ciphertext), "secret") {
		t.Error("the client secret is readable in the stored row")
	}
	t.Logf("stored   : kind=%s keyversion=%d ciphertext=%d bytes", row.Kind, row.KeyVersion, len(row.Ciphertext))

	resolved, err := srv.Resolve(board, []string{"DUENDE"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	header := resolved["$DUENDE"]
	if !strings.HasPrefix(header, "Bearer ey") {
		t.Fatalf("header = %q, want a real bearer JWT", header)
	}
	t.Logf("resolved : %s...", header[:60])

	// The token must have been cached, not re-fetched on every resolve.
	plaintext, err := srv.unseal(&store.rows[0])
	if err != nil {
		t.Fatalf("unseal: %v", err)
	}
	var stored models.OAuth2Material
	_ = json.Unmarshal(plaintext, &stored)

	if stored.AccessToken == "" {
		t.Error("the fetched token was not persisted")
	}
	if stored.ClientSecret != "secret" {
		t.Errorf("the client secret did not survive the round trip: %q", stored.ClientSecret)
	}
	ttl := time.Until(stored.ExpiresAt)
	t.Logf("expires  : in %v (token_type=%q)", ttl.Round(time.Second), stored.TokenType)

	second, err := srv.Resolve(board, []string{"DUENDE"})
	if err != nil {
		t.Fatalf("second resolve: %v", err)
	}
	if second["$DUENDE"] != header {
		t.Error("the second resolve fetched a new token instead of using the cache")
	}
	t.Log("cache    : second resolve reused the stored token")

	// A locked-out worker must not spend a second request.
	locked := oauthService(t, store, nil, &mocks.MockLocker{
		AcquireFn: func(string, time.Duration) (string, bool, error) { return "", false, nil },
	})
	locked.tokens = oauth.New()
	if _, err := locked.Resolve(board, []string{"DUENDE"}); err != nil {
		t.Errorf("a valid cached token should resolve even without the lock: %v", err)
	}
	t.Log("lock     : cached token resolves without touching the provider")
}
