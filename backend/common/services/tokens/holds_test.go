package tokens

import (
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
)

type recordedTokens struct {
	*mocks.MemoryAccessTokenStore
	s *mocks.Steps
}

func (r recordedTokens) ListAccessTokens(user string) ([]models.AccessToken, error) {
	r.s.Add("list")
	return r.MemoryAccessTokenStore.ListAccessTokens(user)
}

func (r recordedTokens) CreateAccessToken(token *models.AccessToken) error {
	r.s.Add("create")
	return r.MemoryAccessTokenStore.CreateAccessToken(token)
}

// The active tokens are counted and the new one stored in the
// user's turn, so parallel requests cannot pass MAX_TOKENS.
func TestCreate_CountsAndStoresInTheUsersTurn(t *testing.T) {
	var s mocks.Steps
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	svc := New(recordedTokens{&mocks.MemoryAccessTokenStore{}, &s}, &mocks.CountingLimiter{}, s.Plans(), func() time.Time { return now })

	if _, err := svc.Create(ana, "laptop", 0); err != nil {
		t.Fatal(err)
	}

	if !s.In("hold "+LIMIT_TOKENS+" user:"+ana.ID, "list", "create") {
		t.Errorf("steps = %v", s.List())
	}
}
