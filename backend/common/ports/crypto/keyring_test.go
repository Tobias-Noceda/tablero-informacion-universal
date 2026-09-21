package crypto

import (
	"bytes"
	"encoding/base64"
	"errors"
	"sync"
	"testing"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

func kekEntry(version int, fill byte) string {
	return string(rune('0'+version)) + ":" + base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{fill}, 32))
}

func keyring(t *testing.T, store infrastructure.KeyStore, kekEntries string) *Keyring {
	t.Helper()
	kek, err := NewFromKeys(kekEntries)
	if err != nil {
		t.Fatalf("kek: %v", err)
	}
	return NewKeyring(kek, store)
}

func TestKeyring_RoundTrip(t *testing.T) {
	store := &mocks.MemoryKeyStore{}
	ring := keyring(t, store, kekEntry(1, 0xA1))
	scope := models.BoardScope(uuid.New())

	sealed, err := ring.Seal(scope, "aad", []byte("the secret"))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if bytes.Contains(sealed.Ciphertext, []byte("the secret")) {
		t.Error("plaintext is visible in the ciphertext")
	}

	plaintext, stale, err := ring.Open(scope, "aad", sealed)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if string(plaintext) != "the secret" || stale {
		t.Errorf("got %q stale=%v", plaintext, stale)
	}

	keys, _ := store.ListKeys()
	if len(keys) != 1 || !keys[0].Active || keys[0].Scope != scope || keys[0].KEKVersion != 1 {
		t.Errorf("stored keys = %+v, want one active key for the scope under KEK 1", keys)
	}
	if bytes.Contains(keys[0].WrappedKey, sealed.Ciphertext) {
		t.Error("the data key store holds ciphertext of a secret")
	}
}

// Every scope gets its own key, so a key for board A can never open what
// belongs to board B, even with the same AAD.
func TestKeyring_ScopesDoNotShareKeys(t *testing.T) {
	store := &mocks.MemoryKeyStore{}
	ring := keyring(t, store, kekEntry(1, 0xA1))
	a := models.BoardScope(uuid.New())
	b := models.BoardScope(uuid.New())

	sealedA, _ := ring.Seal(a, "aad", []byte("for a"))
	if _, err := ring.Seal(b, "aad", []byte("for b")); err != nil {
		t.Fatalf("seal b: %v", err)
	}

	if _, _, err := ring.Open(b, "aad", sealedA); err == nil {
		t.Error("scope b opened a ciphertext sealed for scope a")
	}

	keys, _ := store.ListKeys()
	if len(keys) != 2 || bytes.Equal(keys[0].WrappedKey, keys[1].WrappedKey) {
		t.Errorf("want two distinct keys, got %+v", keys)
	}
}

func TestKeyring_ShredMakesTheScopeUnreadable(t *testing.T) {
	store := &mocks.MemoryKeyStore{}
	ring := keyring(t, store, kekEntry(1, 0xA1))
	scope := models.UserScope("alice")

	sealed, _ := ring.Seal(scope, "aad", []byte("gone"))

	if err := ring.Shred(scope); err != nil {
		t.Fatalf("shred: %v", err)
	}

	if _, _, err := ring.Open(scope, "aad", sealed); err == nil {
		t.Error("a ciphertext was opened after its key was shredded")
	}
}

// Two workers sealing the first secret of a scope at once must end up with
// the same key, or one of the two secrets becomes unreadable.
func TestKeyring_FirstKeyRaceConvergesOnOneKey(t *testing.T) {
	store := &mocks.MemoryKeyStore{}
	ring := keyring(t, store, kekEntry(1, 0xA1))
	scope := models.BoardScope(uuid.New())

	const workers = 16
	sealed := make([]*Sealed, workers)
	var wg sync.WaitGroup
	for i := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s, err := ring.Seal(scope, "aad", []byte("x"))
			if err != nil {
				t.Errorf("seal: %v", err)
				return
			}
			sealed[i] = s
		}()
	}
	wg.Wait()

	keys, _ := store.ListKeys()
	if len(keys) != 1 {
		t.Fatalf("race produced %d keys, want 1", len(keys))
	}
	for _, s := range sealed {
		if s == nil {
			continue
		}
		if s.KeyID != keys[0].Id {
			t.Errorf("a secret was sealed under %s, not the surviving key %s", s.KeyID, keys[0].Id)
		}
		if _, _, err := ring.Open(scope, "aad", s); err != nil {
			t.Errorf("open: %v", err)
		}
	}
}

func TestKeyring_RewrapMovesEveryKeyToTheCurrentKEK(t *testing.T) {
	store := &mocks.MemoryKeyStore{}
	old := keyring(t, store, kekEntry(1, 0xA1))
	scopeA := models.BoardScope(uuid.New())
	scopeB := models.SystemScope

	sealedA, _ := old.Seal(scopeA, "a", []byte("for a"))
	sealedB, _ := old.Seal(scopeB, "b", []byte("for b"))

	rotated := keyring(t, store, kekEntry(1, 0xA1)+","+kekEntry(2, 0xB2))
	rewrapped, err := rotated.Rewrap()
	if err != nil {
		t.Fatalf("rewrap: %v", err)
	}
	if rewrapped != 2 {
		t.Errorf("rewrapped %d keys, want 2", rewrapped)
	}

	keys, _ := store.ListKeys()
	for _, key := range keys {
		if key.KEKVersion != 2 {
			t.Errorf("key %s still under KEK %d", key.Id, key.KEKVersion)
		}
	}

	// Once nothing depends on version 1 it can be dropped from the ring.
	fresh := keyring(t, store, kekEntry(2, 0xB2))
	for _, c := range []struct {
		scope  models.SecretScope
		aad    string
		sealed *Sealed
		want   string
	}{{scopeA, "a", sealedA, "for a"}, {scopeB, "b", sealedB, "for b"}} {
		plaintext, _, err := fresh.Open(c.scope, c.aad, c.sealed)
		if err != nil || string(plaintext) != c.want {
			t.Errorf("open after rewrap: %q, %v", plaintext, err)
		}
	}

	// A second run finds nothing to do.
	if again, _ := rotated.Rewrap(); again != 0 {
		t.Errorf("second rewrap touched %d keys, want 0", again)
	}
}

func TestKeyring_RotateKeepsOldReadableAndSealsWithNew(t *testing.T) {
	store := &mocks.MemoryKeyStore{}
	ring := keyring(t, store, kekEntry(1, 0xA1))
	scope := models.BoardScope(uuid.New())

	before, _ := ring.Seal(scope, "aad", []byte("old"))

	fresh, err := ring.Rotate(scope)
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if !fresh.Active || fresh.Id == before.KeyID {
		t.Errorf("rotate returned %+v, want a new active key", fresh)
	}

	plaintext, stale, err := ring.Open(scope, "aad", before)
	if err != nil || string(plaintext) != "old" {
		t.Fatalf("open old: %q, %v", plaintext, err)
	}
	if !stale {
		t.Error("a ciphertext under a retired key was not reported stale")
	}

	after, _ := ring.Seal(scope, "aad", []byte("new"))
	if after.KeyID != fresh.Id {
		t.Errorf("sealed with %s, want the new key %s", after.KeyID, fresh.Id)
	}
	if _, stale, _ := ring.Open(scope, "aad", after); stale {
		t.Error("a ciphertext under the active key was reported stale")
	}

	keys, _ := store.ListKeys()
	if len(keys) != 2 {
		t.Fatalf("want the retired and the active key, got %d", len(keys))
	}
	for _, key := range keys {
		if key.Id == before.KeyID && (key.Active || key.RetiredAt == nil) {
			t.Errorf("old key not retired: %+v", key)
		}
	}
}

// Rotating a scope that has never sealed anything simply provisions its key.
func TestKeyring_RotateFreshScope(t *testing.T) {
	store := &mocks.MemoryKeyStore{}
	ring := keyring(t, store, kekEntry(1, 0xA1))

	if _, err := ring.Rotate(models.SystemScope); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	keys, _ := store.ListKeys()
	if len(keys) != 1 || !keys[0].Active {
		t.Errorf("keys = %+v, want one active key", keys)
	}
}

func TestKeyring_PruneDropsOnlyRetiredKeys(t *testing.T) {
	store := &mocks.MemoryKeyStore{}
	ring := keyring(t, store, kekEntry(1, 0xA1))
	scope := models.BoardScope(uuid.New())
	other := models.BoardScope(uuid.New())

	_, _ = ring.Seal(scope, "aad", []byte("x"))
	_, _ = ring.Seal(other, "aad", []byte("y"))
	fresh, _ := ring.Rotate(scope)

	if err := ring.Prune(scope); err != nil {
		t.Fatalf("prune: %v", err)
	}

	keys, _ := store.ListKeys()
	if len(keys) != 2 {
		t.Fatalf("keys = %+v, want the active key of each scope", keys)
	}
	for _, key := range keys {
		if key.Scope == scope && key.Id != fresh.Id {
			t.Errorf("retired key survived: %+v", key)
		}
	}
}

// A ciphertext whose key belongs to another scope must not open, even if the
// key id is known: the id is not a secret.
func TestKeyring_OpenRejectsKeyFromAnotherScope(t *testing.T) {
	store := &mocks.MemoryKeyStore{}
	ring := keyring(t, store, kekEntry(1, 0xA1))
	a := models.BoardScope(uuid.New())

	sealed, _ := ring.Seal(a, "aad", []byte("x"))

	_, _, err := ring.Open(models.UserScope("mallory"), "aad", sealed)
	if !errors.Is(err, infrastructure.ErrKeyNotFound) {
		t.Errorf("got %v, want ErrKeyNotFound", err)
	}
}
