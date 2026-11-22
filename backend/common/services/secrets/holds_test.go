package secrets

import (
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

type recordedSecrets struct {
	*memoryStore
	s *mocks.Steps
}

func (r recordedSecrets) UpsertSecret(secret *models.Secret) error {
	r.s.Add("upsert")
	return r.memoryStore.UpsertSecret(secret)
}

// The payer's room for one more secret is checked and the secret
// stored in the payer's turn.
func TestPut_ChecksRoomAndStoresInThePayersTurn(t *testing.T) {
	var s mocks.Steps
	store := newStore()
	plans := s.Plans()
	srv := serviceOn(t, store, plans)
	srv.store = recordedSecrets{store, &s}
	scope := models.BoardScope(uuid.New())
	payer, _ := plans.ForSecretScope(scope)

	if err := srv.Put(scope, owner, "API_KEY", models.SecretApiKey, "value"); err != nil {
		t.Fatal(err)
	}

	if !s.In("hold "+models.LimitSecrets+" user:"+payer.Payer, "upsert") {
		t.Errorf("steps = %v", s.List())
	}
}

// Rewriting a secret takes no new room, so it needs no turn.
func TestPut_ARewriteNeedsNoTurn(t *testing.T) {
	store := newStore()
	scope := models.BoardScope(uuid.New())
	if err := service(t, store).Put(scope, owner, "API_KEY", models.SecretApiKey, "first"); err != nil {
		t.Fatal(err)
	}

	var s mocks.Steps
	srv := serviceOn(t, store, s.Plans())
	if err := srv.Put(scope, owner, "API_KEY", models.SecretApiKey, "second"); err != nil {
		t.Fatal(err)
	}
	for _, step := range s.List() {
		if step != "release" && len(step) > 4 && step[:4] == "hold" {
			t.Errorf("a rewrite took a turn: %v", s.List())
		}
	}
}
