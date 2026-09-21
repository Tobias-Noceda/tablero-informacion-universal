package secrets

import (
	"testing"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

func activeKeyID(t *testing.T, store *memoryStore, scope models.SecretScope) uuid.UUID {
	t.Helper()
	key, err := store.keys.FindActiveKey(scope)
	if err != nil {
		t.Fatalf("active key: %v", err)
	}
	return key.Id
}

func TestReseal_MigratesEverySecretAndPrunesTheOldKey(t *testing.T) {
	store := newStore()
	srv := service(t, store)
	board := models.BoardScope(uuid.New())
	other := models.BoardScope(uuid.New())

	_ = srv.Put(board, owner, "ONE", models.SecretApiKey, "1")
	_ = srv.Put(board, owner, "TWO", models.SecretBearer, "2")
	_ = srv.Put(other, owner, "ELSEWHERE", models.SecretApiKey, "3")
	oldKey := activeKeyID(t, store, board)

	if _, err := srv.keyring.Rotate(board); err != nil {
		t.Fatalf("rotate: %v", err)
	}

	resealed, err := srv.Reseal(board)
	if err != nil {
		t.Fatalf("reseal: %v", err)
	}
	if resealed != 2 {
		t.Errorf("resealed %d secrets, want 2", resealed)
	}

	newKey := activeKeyID(t, store, board)
	for _, row := range store.rows {
		if row.Scope == board && row.KeyID != newKey {
			t.Errorf("%s still sealed under %s", row.Name, row.KeyID)
		}
		if row.Scope == other && row.KeyID == newKey {
			t.Errorf("%s was resealed although it belongs to another scope", row.Name)
		}
	}

	if _, err := store.keys.FindKey(oldKey); err == nil {
		t.Error("the retired key survived the reseal")
	}

	resolved, err := srv.Resolve(board, []string{"ONE", "TWO"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved["$ONE"] != "1" || resolved["$TWO"] != "Bearer 2" {
		t.Errorf("resolved = %v", resolved)
	}
}

// A secret still under a retired key is served and quietly moved to the
// active one, so the key can eventually be dropped without a batch.
func TestResolve_MigratesASecretUnderARetiredKey(t *testing.T) {
	store := newStore()
	srv := service(t, store)
	board := models.BoardScope(uuid.New())

	_ = srv.Put(board, owner, "API_KEY", models.SecretApiKey, "abc123")
	if _, err := srv.keyring.Rotate(board); err != nil {
		t.Fatalf("rotate: %v", err)
	}

	resolved, err := srv.Resolve(board, []string{"API_KEY"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved["$API_KEY"] != "abc123" {
		t.Errorf("resolved = %v", resolved)
	}

	if store.rows[0].KeyID != activeKeyID(t, store, board) {
		t.Error("the secret was not migrated to the active key")
	}
}

func TestPurge_LeavesNoSecretsAndNoKeys(t *testing.T) {
	store := newStore()
	srv := service(t, store)
	board := models.BoardScope(uuid.New())
	other := models.BoardScope(uuid.New())

	_ = srv.Put(board, owner, "ONE", models.SecretApiKey, "1")
	_ = srv.Put(other, owner, "KEEP", models.SecretApiKey, "2")

	if err := srv.Purge(board); err != nil {
		t.Fatalf("purge: %v", err)
	}

	for _, row := range store.rows {
		if row.Scope == board {
			t.Errorf("secret %s survived the purge", row.Name)
		}
	}
	if _, err := store.keys.FindActiveKey(board); err == nil {
		t.Error("the purged scope still has a key")
	}
	if _, err := store.keys.FindActiveKey(other); err != nil {
		t.Error("another scope lost its key")
	}
}

func TestRotateScope_EndsWithEverythingUnderOneFreshKey(t *testing.T) {
	store := newStore()
	srv := service(t, store)
	board := models.BoardScope(uuid.New())

	_ = srv.Put(board, owner, "API_KEY", models.SecretApiKey, "abc123")
	before := activeKeyID(t, store, board)

	if err := srv.RotateScope(board); err != nil {
		t.Fatalf("rotate: %v", err)
	}

	keys, _ := store.keys.ListKeys()
	if len(keys) != 1 || keys[0].Id == before || !keys[0].Active {
		t.Errorf("keys = %+v, want exactly one fresh active key", keys)
	}
	if store.rows[0].KeyID != keys[0].Id {
		t.Error("the secret is not under the fresh key")
	}
}

func TestRotateAll_TouchesEveryScopeWithAKey(t *testing.T) {
	store := newStore()
	srv := service(t, store)
	a := models.BoardScope(uuid.New())
	b := models.UserScope("alice")

	_ = srv.Put(a, owner, "A", models.SecretApiKey, "1")
	_ = srv.Put(b, models.Principal{ID: "alice"}, "B", models.SecretApiKey, "2")
	_ = srv.Put(models.SystemScope, anonymous, "S", models.SecretApiKey, "3")
	before := map[uuid.UUID]bool{}
	for _, scope := range []models.SecretScope{a, b, models.SystemScope} {
		before[activeKeyID(t, store, scope)] = true
	}

	if err := srv.RotateAll(); err != nil {
		t.Fatalf("rotate all: %v", err)
	}

	keys, _ := store.keys.ListKeys()
	if len(keys) != 3 {
		t.Fatalf("keys = %d, want 3", len(keys))
	}
	for _, key := range keys {
		if before[key.Id] {
			t.Errorf("scope %s kept its old key", key.Scope.Key())
		}
	}
	for _, row := range store.rows {
		if before[row.KeyID] {
			t.Errorf("%s is still under an old key", row.Name)
		}
	}
}

func TestListKeys_NeverCarriesMaterial(t *testing.T) {
	store := newStore()
	srv := service(t, store)
	_ = srv.Put(models.SystemScope, anonymous, "S", models.SecretApiKey, "3")

	keys, err := srv.ListKeys(anonymous)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(keys) != 1 || keys[0].Scope != models.SystemScope {
		t.Fatalf("keys = %+v", keys)
	}
	if keys[0].WrappedKey != nil || keys[0].Nonce != nil {
		t.Error("key material was returned")
	}
}
