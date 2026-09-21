//go:build integration

package mongo

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/ports/crypto"
	secretsrv "github.com/Secreto31126/tesis/common/services/secrets"
	"github.com/google/uuid"
)

// Runs against the Mongo named by MONGODB_URI / MONGO_DATABASE (see
// run_backend_integration.*). Every test works in its own scopes, and the
// collections are dropped at the end.
func integrationDB(t *testing.T) *MongoDB {
	t.Helper()

	db, err := New()
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := db.client.Ping(context.Background(), nil); err != nil {
		t.Skipf("mongo not reachable: %v", err)
	}
	if err := db.EnsureIndexes(); err != nil {
		t.Fatalf("indexes: %v", err)
	}

	t.Cleanup(func() {
		ctx, cancel := timeout()
		defer cancel()
		_ = db.secrets.Drop(ctx)
		_ = db.dataKeys.Drop(ctx)
		_ = db.groups.Drop(ctx)
		_ = db.users.Drop(ctx)
		_ = db.Close()
	})

	return db
}

func integrationService(t *testing.T, db *MongoDB, kekEntries string) *secretsrv.SecretsService {
	t.Helper()
	kek, err := crypto.NewFromKeys(kekEntries)
	if err != nil {
		t.Fatalf("kek: %v", err)
	}
	return secretsrv.New(db, &mocks.MockScopePolicy{}, crypto.NewKeyring(kek, db),
		&mocks.MockTokenClient{}, &mocks.MockLocker{}, &mocks.MockHandshakeStore{}, db)
}

func kek(version int, fill byte) string {
	return string(rune('0'+version)) + ":" + base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{fill}, 32))
}

func TestMongo_OneActiveKeyPerScope(t *testing.T) {
	db := integrationDB(t)
	scope := models.BoardScope(uuid.New())

	first := &models.DataKey{Id: uuid.New(), Scope: scope, WrappedKey: []byte("w"), Nonce: []byte("n"), KEKVersion: 1, Active: true, CreatedAt: time.Now()}
	if err := db.InsertKey(first); err != nil {
		t.Fatalf("first insert: %v", err)
	}

	second := &models.DataKey{Id: uuid.New(), Scope: scope, WrappedKey: []byte("w"), Nonce: []byte("n"), KEKVersion: 1, Active: true, CreatedAt: time.Now()}
	if err := db.InsertKey(second); !errors.Is(err, infrastructure.ErrActiveKeyExists) {
		t.Fatalf("second active insert: %v, want ErrActiveKeyExists", err)
	}

	if err := db.RetireKey(first.Id, time.Now()); err != nil {
		t.Fatalf("retire: %v", err)
	}
	if err := db.InsertKey(second); err != nil {
		t.Fatalf("insert after retiring: %v", err)
	}

	found, err := db.FindActiveKey(scope)
	if err != nil || found.Id != second.Id {
		t.Errorf("active = %+v, %v; want the second key", found, err)
	}
}

func TestMongo_ConcurrentFirstSecretsShareOneKey(t *testing.T) {
	db := integrationDB(t)
	srv := integrationService(t, db, kek(1, 0xA1))
	scope := models.BoardScope(uuid.New())
	principal := models.Principal{ID: "anyone"}

	const workers = 12
	var wg sync.WaitGroup
	for i := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			name := "KEY_" + string(rune('A'+i))
			if err := srv.Put(scope, principal, name, models.SecretApiKey, "v"); err != nil {
				t.Errorf("%s: %v", name, err)
			}
		}()
	}
	wg.Wait()

	keys, _ := db.ListKeys()
	active := 0
	for _, key := range keys {
		if key.Scope == scope && key.Active {
			active++
		}
	}
	if active != 1 {
		t.Fatalf("%d active keys for the scope, want 1", active)
	}

	stored, _ := db.ListSecrets(scope)
	if len(stored) != workers {
		t.Fatalf("stored %d secrets, want %d", len(stored), workers)
	}
	names := make([]string, 0, workers)
	for _, s := range stored {
		names = append(names, s.Name)
	}
	resolved, err := srv.Resolve(scope, names)
	if err != nil || len(resolved) != workers {
		t.Errorf("resolve: %d values, %v", len(resolved), err)
	}
}

func TestMongo_UniqueSecretPerScopeAndName(t *testing.T) {
	db := integrationDB(t)
	srv := integrationService(t, db, kek(1, 0xA1))
	scope := models.UserScope("alice")
	principal := models.Principal{ID: "alice"}

	_ = srv.Put(scope, principal, "TOKEN", models.SecretApiKey, "one")
	_ = srv.Put(scope, principal, "TOKEN", models.SecretApiKey, "two")

	stored, _ := db.ListSecrets(scope)
	if len(stored) != 1 {
		t.Fatalf("stored %d rows, want the upsert to replace", len(stored))
	}
	resolved, _ := srv.Resolve(scope, []string{"TOKEN"})
	if resolved["$TOKEN"] != "two" {
		t.Errorf("resolved %v, want the latest value", resolved)
	}

	other, _ := srv.Resolve(models.BoardScope(uuid.New()), []string{"TOKEN"})
	if len(other) != 0 {
		t.Error("another scope saw the secret")
	}
}

func TestMongo_RotationEndToEnd(t *testing.T) {
	db := integrationDB(t)
	old := integrationService(t, db, kek(1, 0xA1))
	board := models.BoardScope(uuid.New())
	principal := models.Principal{ID: "anyone"}

	_ = old.Put(board, principal, "A", models.SecretApiKey, "a")
	_ = old.Put(models.SystemScope, principal, "S", models.SecretApiKey, "s")

	// Master key rotation: every data key moves to version 2, secrets untouched.
	current := integrationService(t, db, kek(1, 0xA1)+","+kek(2, 0xB2))
	if n, err := current.Rewrap(); err != nil || n != 2 {
		t.Fatalf("rewrap: %d, %v", n, err)
	}
	keys, _ := db.ListKeys()
	for _, key := range keys {
		if key.KEKVersion != 2 {
			t.Errorf("key %s still under KEK %d", key.Id, key.KEKVersion)
		}
	}

	// Data key rotation of one scope: secrets migrate, the retired key is gone.
	before, _ := db.FindActiveKey(board)
	if err := current.RotateScope(board); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if _, err := db.FindKey(before.Id); !errors.Is(err, infrastructure.ErrKeyNotFound) {
		t.Errorf("retired key still present: %v", err)
	}

	// Only the new master key is left; everything must still open.
	fresh := integrationService(t, db, kek(2, 0xB2))
	for _, c := range []struct {
		scope models.SecretScope
		name  string
		want  string
	}{{board, "A", "a"}, {models.SystemScope, "S", "s"}} {
		resolved, err := fresh.Resolve(c.scope, []string{c.name})
		if err != nil || resolved["$"+c.name] != c.want {
			t.Errorf("%s: %v, %v", c.name, resolved, err)
		}
	}

	// Rotate everything and confirm no scope kept its key.
	previous := map[uuid.UUID]bool{}
	for _, key := range keys {
		previous[key.Id] = true
	}
	if err := fresh.RotateAll(); err != nil {
		t.Fatalf("rotate all: %v", err)
	}
	after, _ := db.ListKeys()
	for _, key := range after {
		if previous[key.Id] {
			t.Errorf("scope %s kept key %s", key.Scope.Key(), key.Id)
		}
	}
}

func TestMongo_PurgeRemovesSecretsAndKeys(t *testing.T) {
	db := integrationDB(t)
	srv := integrationService(t, db, kek(1, 0xA1))
	board := models.BoardScope(uuid.New())
	other := models.BoardScope(uuid.New())
	principal := models.Principal{ID: "anyone"}

	_ = srv.Put(board, principal, "A", models.SecretApiKey, "a")
	_ = srv.Put(other, principal, "B", models.SecretApiKey, "b")

	if err := srv.Purge(board); err != nil {
		t.Fatalf("purge: %v", err)
	}

	if stored, _ := db.ListSecrets(board); len(stored) != 0 {
		t.Errorf("%d secrets survived the purge", len(stored))
	}
	if _, err := db.FindActiveKey(board); !errors.Is(err, infrastructure.ErrKeyNotFound) {
		t.Errorf("key survived the purge: %v", err)
	}
	if resolved, _ := srv.Resolve(other, []string{"B"}); resolved["$B"] != "b" {
		t.Error("another scope was affected")
	}
}
