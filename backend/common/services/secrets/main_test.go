package secrets

import (
	"bytes"
	"encoding/base64"
	"errors"
	"github.com/Secreto31126/tesis/common/infrastructure"
	"slices"
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
	rows   []models.Secret
	keys   *mocks.MemoryKeyStore
	groups *mocks.MemoryGroupStore
}

func newStore() *memoryStore {
	s := &memoryStore{keys: &mocks.MemoryKeyStore{}, groups: &mocks.MemoryGroupStore{}}
	// Mirrors the Mongo upsert: one row per board + name, replaced in place,
	// grants untouched.
	s.UpsertSecretFn = func(secret *models.Secret) error {
		for i, row := range s.rows {
			if row.Scope == secret.Scope && row.Name == secret.Name {
				grants := row.Grants
				s.rows[i] = *secret
				s.rows[i].Grants = grants
				return nil
			}
		}
		s.rows = append(s.rows, *secret)
		return nil
	}
	s.SetGrantsFn = func(scope models.SecretScope, name string, grants []models.Grant) error {
		for i, row := range s.rows {
			if row.Scope == scope && row.Name == name {
				s.rows[i].Grants = grants
				return nil
			}
		}
		return infrastructure.ErrUnknownCredential
	}
	s.FindGrantedFn = func(audiences []models.Audience) ([]models.Secret, error) {
		var out []models.Secret
		for _, row := range s.rows {
			for _, grant := range row.Grants {
				if slices.Contains(audiences, grant.To) {
					out = append(out, row)
					break
				}
			}
		}
		return out, nil
	}
	s.FindSecretsFn = func(scope models.SecretScope, names []string) ([]models.Secret, error) {
		var out []models.Secret
		for _, row := range s.rows {
			if row.Scope != scope {
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
	s.ListSecretsFn = func(scope models.SecretScope) ([]models.Secret, error) {
		var out []models.Secret
		for _, row := range s.rows {
			if row.Scope == scope {
				out = append(out, row)
			}
		}
		return out, nil
	}
	s.DeleteSecretsFn = func(scope models.SecretScope) error {
		kept := s.rows[:0]
		for _, row := range s.rows {
			if row.Scope != scope {
				kept = append(kept, row)
			}
		}
		s.rows = kept
		return nil
	}
	return s
}

var (
	owner        = models.Principal{ID: "owner-cognito-id"}
	collaborator = models.Principal{ID: "collab-cognito-id"}
	stranger     = models.Principal{ID: "stranger"}
	anonymous    = models.Principal{}
)

func service(t *testing.T, store *memoryStore) *SecretsService {
	t.Helper()
	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x2B}, 32))
	sealer, err := crypto.NewFromKeys("1:" + key)
	if err != nil {
		t.Fatalf("sealer: %v", err)
	}
	boards := &mocks.MockDB{
		FindBoardFn: func(id uuid.UUID) (*models.Board, error) {
			return &models.Board{Id: id, Owner: owner.ID, Collaborators: []string{collaborator.ID}}, nil
		},
	}
	return New(store, NewPolicy(boards, store.groups), crypto.NewKeyring(sealer, store.keys), &mocks.MockTokenClient{}, &mocks.MockLocker{}, &mocks.MockHandshakeStore{}, store.groups)
}

func TestPut_StoresOnlyCiphertext(t *testing.T) {
	store := newStore()
	board := models.BoardScope(uuid.New())
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
	if len(row.Nonce) == 0 || row.KeyID == uuid.Nil {
		t.Errorf("nonce/key not recorded: %+v", row)
	}
}

func TestResolve_RoundTripKeyedForParams(t *testing.T) {
	store := newStore()
	board := models.BoardScope(uuid.New())
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
	ownerBoard := models.BoardScope(uuid.New())
	srv := service(t, store)

	if err := srv.Put(ownerBoard, owner, "API_KEY", models.SecretApiKey, "abc123"); err != nil {
		t.Fatalf("put: %v", err)
	}

	resolved, err := srv.Resolve(models.BoardScope(uuid.New()), []string{"API_KEY"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(resolved) != 0 {
		t.Errorf("another board resolved %v", resolved)
	}
}

func TestList_NeverCarriesTheValue(t *testing.T) {
	store := newStore()
	board := models.BoardScope(uuid.New())
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
	board := models.BoardScope(uuid.New())
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
	board := models.BoardScope(uuid.New())

	for _, caller := range []models.Principal{collaborator, stranger, anonymous} {
		err := srv.Put(board, caller, "API_KEY", models.SecretApiKey, "v")
		if !errors.Is(err, ErrForbidden) {
			t.Errorf("caller %q got %v, want ErrForbidden", caller.ID, err)
		}
	}

	if len(store.rows) != 0 {
		t.Error("a non-owner managed to write a secret")
	}
}

func TestDelete_OwnerOnly(t *testing.T) {
	store := newStore()
	deleted := false
	store.DeleteSecretFn = func(models.SecretScope, string) error {
		deleted = true
		return nil
	}
	srv := service(t, store)
	board := models.BoardScope(uuid.New())

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
	board := models.BoardScope(uuid.New())

	if _, err := srv.List(board, collaborator); err != nil {
		t.Errorf("collaborator got %v, want nil", err)
	}

	if _, err := srv.List(board, stranger); !errors.Is(err, ErrForbidden) {
		t.Errorf("stranger got %v, want ErrForbidden", err)
	}
}

// The stored kind decides the wire form, so the same secret can serve an API
// that wants a raw header and one that wants an Authorization scheme.
func TestResolve_ShapesTheValuePerKind(t *testing.T) {
	cases := []struct {
		kind  models.SecretKind
		value string
		want  string
	}{
		{models.SecretApiKey, "cur_live_abc123", "cur_live_abc123"},
		{models.SecretBearer, "ey.jwt.token", "Bearer ey.jwt.token"},
		// base64("alice:s3cr3t")
		{models.SecretBasic, "alice:s3cr3t", "Basic YWxpY2U6czNjcjN0"},
	}

	for _, c := range cases {
		store := newStore()
		srv := service(t, store)
		board := models.BoardScope(uuid.New())

		if err := srv.Put(board, owner, "CRED", c.kind, c.value); err != nil {
			t.Fatalf("%s: put: %v", c.kind, err)
		}

		resolved, err := srv.Resolve(board, []string{"CRED"})
		if err != nil {
			t.Fatalf("%s: resolve: %v", c.kind, err)
		}
		if resolved["$CRED"] != c.want {
			t.Errorf("%s resolved to %q, want %q", c.kind, resolved["$CRED"], c.want)
		}
	}
}

// Whatever the wire form, what gets persisted is the raw credential.
func TestPut_StoresTheRawValueNotTheWireForm(t *testing.T) {
	store := newStore()
	srv := service(t, store)
	board := models.BoardScope(uuid.New())

	if err := srv.Put(board, owner, "CRED", models.SecretBearer, "ey.jwt.token"); err != nil {
		t.Fatalf("put: %v", err)
	}

	plaintext, err := srv.unseal(&store.rows[0])
	if err != nil {
		t.Fatalf("unseal: %v", err)
	}
	if string(plaintext) != "ey.jwt.token" {
		t.Errorf("stored %q, want the raw token without the Bearer prefix", plaintext)
	}
}

// The same owner string under another kind is a different scope entirely.
func TestResolve_IsScopedToItsKind(t *testing.T) {
	store := newStore()
	srv := service(t, store)
	id := uuid.New()

	if err := srv.Put(models.BoardScope(id), owner, "API_KEY", models.SecretApiKey, "abc123"); err != nil {
		t.Fatalf("put: %v", err)
	}

	resolved, err := srv.Resolve(models.UserScope(id.String()), []string{"API_KEY"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(resolved) != 0 {
		t.Errorf("a user scope resolved a board secret: %v", resolved)
	}
}

func TestListSystem_ReportsKnownNamesAndExtras(t *testing.T) {
	store := newStore()
	srv := service(t, store)

	if err := srv.Put(models.SystemScope, anonymous, "EXTRA_KEY", models.SecretApiKey, "v"); err != nil {
		t.Fatalf("put: %v", err)
	}

	statuses, err := srv.ListSystem(anonymous)
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	byName := make(map[string]models.SystemSecretStatus)
	for _, s := range statuses {
		byName[s.Name] = s
	}

	nasa := byName[string(models.SystemNasaApiKey)]
	if !nasa.Known || nasa.Configured {
		t.Errorf("NASA_API_KEY = %+v, want known and not configured", nasa)
	}

	extra := byName["EXTRA_KEY"]
	if extra.Known || !extra.Configured || extra.Kind != models.SecretApiKey {
		t.Errorf("EXTRA_KEY = %+v, want configured, not known, api_key", extra)
	}
}

func TestMissingSystemSecrets_ListsWhatTheCodeExpectsButIsNotStored(t *testing.T) {
	store := newStore()
	srv := service(t, store)

	missing, err := srv.MissingSystemSecrets()
	if err != nil {
		t.Fatalf("missing: %v", err)
	}
	if len(missing) != len(models.KnownSystemSecrets) {
		t.Fatalf("missing = %v, want every known secret before any is provisioned", missing)
	}

	_ = srv.Put(models.SystemScope, anonymous, string(models.SystemNasaApiKey), models.SecretApiKey, "v")

	missing, _ = srv.MissingSystemSecrets()
	for _, name := range missing {
		if name == models.SystemNasaApiKey {
			t.Error("a provisioned secret was reported missing")
		}
	}
}
