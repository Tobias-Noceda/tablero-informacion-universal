package secrets

import (
	"bytes"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/ports/crypto"
	"github.com/google/uuid"
)

// memoryStore keeps whatever Put wrote so Resolve can read it back.
type memoryStore struct {
	mocks.MockSecretStore
	rows []models.Secret
}

func newStore() *memoryStore {
	s := &memoryStore{}
	s.UpsertSecretFn = func(secret *models.Secret) error {
		s.rows = append(s.rows, *secret)
		return nil
	}
	s.FindSecretsFn = func(board uuid.UUID, names []string) ([]models.Secret, error) {
		var out []models.Secret
		for _, row := range s.rows {
			if row.Board != board {
				continue
			}
			for _, name := range names {
				if row.Name == name {
					out = append(out, row)
				}
			}
		}
		return out, nil
	}
	s.ListSecretsFn = func(board uuid.UUID) ([]models.Secret, error) {
		return s.rows, nil
	}
	return s
}

const owner = "owner-cognito-id"

const collaborator = "collab-cognito-id"

func service(t *testing.T, store *memoryStore) *SecretsService {
	t.Helper()
	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x2B}, 32))
	sealer, err := crypto.NewFromKeys("1:" + key)
	if err != nil {
		t.Fatalf("sealer: %v", err)
	}
	boards := &mocks.MockDB{
		FindBoardFn: func(id uuid.UUID) (*models.Board, error) {
			return &models.Board{Id: id, Owner: owner, Collaborators: []string{collaborator}}, nil
		},
	}
	return New(store, boards, sealer)
}

func TestPut_StoresOnlyCiphertext(t *testing.T) {
	store := newStore()
	board := uuid.New()
	srv := service(t, store)

	if err := srv.Put(board, owner, "TICKETMASTER_KEY", models.SecretApiKey, "kpGJZiOXIoaB"); err != nil {
		t.Fatalf("put: %v", err)
	}

	if len(store.rows) != 1 {
		t.Fatalf("stored %d rows, want 1", len(store.rows))
	}
	row := store.rows[0]
	if bytes.Contains(row.Ciphertext, []byte("kpGJZiOXIoaB")) {
		t.Error("the plaintext is present in what was persisted")
	}
	if len(row.Nonce) == 0 || row.KeyVersion != 1 {
		t.Errorf("nonce/version not recorded: %+v", row)
	}
}

func TestResolve_RoundTripKeyedForParams(t *testing.T) {
	store := newStore()
	board := uuid.New()
	srv := service(t, store)

	if err := srv.Put(board, owner, "API_KEY", models.SecretApiKey, "abc123"); err != nil {
		t.Fatalf("put: %v", err)
	}

	resolved, err := srv.Resolve(board, []string{"API_KEY"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved["$API_KEY"] != "abc123" {
		t.Errorf("got %v, want $API_KEY=abc123", resolved)
	}
}

// A secret belongs to one board. Asking from another must not decrypt it.
func TestResolve_IsScopedToItsBoard(t *testing.T) {
	store := newStore()
	ownerBoard := uuid.New()
	srv := service(t, store)

	if err := srv.Put(ownerBoard, owner, "API_KEY", models.SecretApiKey, "abc123"); err != nil {
		t.Fatalf("put: %v", err)
	}

	resolved, err := srv.Resolve(uuid.New(), []string{"API_KEY"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(resolved) != 0 {
		t.Errorf("another board resolved %v", resolved)
	}
}

func TestList_NeverCarriesTheValue(t *testing.T) {
	store := newStore()
	board := uuid.New()
	srv := service(t, store)

	_ = srv.Put(board, owner, "API_KEY", models.SecretApiKey, "abc123")

	metas, err := srv.List(board, owner)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(metas) != 1 || metas[0].Name != "API_KEY" {
		t.Fatalf("got %+v", metas)
	}
	if strings.Contains(strings.Join([]string{metas[0].Name, string(metas[0].Kind)}, "|"), "abc123") {
		t.Error("the value leaked into the metadata")
	}
}

func TestPut_Rejects(t *testing.T) {
	store := newStore()
	board := uuid.New()
	srv := service(t, store)

	cases := []struct {
		label string
		name  string
		kind  models.SecretKind
		value string
	}{
		{"lower case name", "api_key", models.SecretApiKey, "v"},
		{"leading digit", "1KEY", models.SecretApiKey, "v"},
		{"dollar in name", "$KEY", models.SecretApiKey, "v"},
		{"empty name", "", models.SecretApiKey, "v"},
		{"empty value", "KEY", models.SecretApiKey, ""},
		{"unknown kind", "KEY", models.SecretKind("oauth2"), "v"},
		{"oversized", "KEY", models.SecretApiKey, strings.Repeat("x", MAX_SECRET_SIZE+1)},
	}

	for _, c := range cases {
		if err := srv.Put(board, owner, c.name, c.kind, c.value); err == nil {
			t.Errorf("%s: expected rejection", c.label)
		}
	}

	if len(store.rows) != 0 {
		t.Errorf("a rejected secret was still persisted: %+v", store.rows)
	}
}

func TestPut_OwnerOnly(t *testing.T) {
	store := newStore()
	srv := service(t, store)
	board := uuid.New()

	for _, caller := range []string{collaborator, "stranger", ""} {
		err := srv.Put(board, caller, "API_KEY", models.SecretApiKey, "v")
		if !errors.Is(err, ErrForbidden) {
			t.Errorf("caller %q got %v, want ErrForbidden", caller, err)
		}
	}

	if len(store.rows) != 0 {
		t.Error("a non-owner managed to write a secret")
	}
}

func TestDelete_OwnerOnly(t *testing.T) {
	store := newStore()
	deleted := false
	store.DeleteSecretFn = func(_ uuid.UUID, _ string) error {
		deleted = true
		return nil
	}
	srv := service(t, store)
	board := uuid.New()

	if err := srv.Delete(board, collaborator, "API_KEY"); !errors.Is(err, ErrForbidden) {
		t.Errorf("collaborator got %v, want ErrForbidden", err)
	}
	if deleted {
		t.Error("a collaborator deleted a secret")
	}

	if err := srv.Delete(board, owner, "API_KEY"); err != nil {
		t.Errorf("owner got %v, want nil", err)
	}
	if !deleted {
		t.Error("the owner's delete did not reach the store")
	}
}

// A collaborator may know which credentials exist, without reading them.
func TestList_AllowsCollaboratorButNotStrangers(t *testing.T) {
	store := newStore()
	srv := service(t, store)
	board := uuid.New()

	if _, err := srv.List(board, collaborator); err != nil {
		t.Errorf("collaborator got %v, want nil", err)
	}

	if _, err := srv.List(board, "stranger"); !errors.Is(err, ErrForbidden) {
		t.Errorf("stranger got %v, want ErrForbidden", err)
	}
}
