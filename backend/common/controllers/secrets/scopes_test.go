package secrets

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
)

func TestUserSecrets_OwnerManagesTheirOwn(t *testing.T) {
	var stored *models.Secret
	store := &mocks.MockSecretStore{
		UpsertSecretFn: func(secret *models.Secret) error {
			stored = secret
			return nil
		},
	}
	r := setupRouter(store)

	w := do(r, http.MethodPut, "/users/alice/secrets",
		`{"cognito_id":"alice","name":"API_KEY","kind":"api_key","value":"s3cr3t"}`)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (body: %s)", w.Code, w.Body.String())
	}
	if stored == nil || stored.Scope != models.UserScope("alice") {
		t.Fatalf("stored = %+v, want a user-scoped secret", stored)
	}
}

// The path names the scope, the body names the caller; they must agree.
func TestUserSecrets_AnotherUserIsNotFound(t *testing.T) {
	written := false
	store := &mocks.MockSecretStore{
		UpsertSecretFn: func(*models.Secret) error {
			written = true
			return nil
		},
	}
	r := setupRouter(store)

	w := do(r, http.MethodPut, "/users/alice/secrets",
		`{"cognito_id":"bob","name":"API_KEY","kind":"api_key","value":"v"}`)
	if w.Code != http.StatusNotFound {
		t.Errorf("PUT status = %d, want 404", w.Code)
	}
	if written {
		t.Error("another user's write reached the store")
	}

	w = do(r, http.MethodGet, "/users/alice/secrets?cognito_id=bob", "")
	if w.Code != http.StatusNotFound {
		t.Errorf("GET status = %d, want 404", w.Code)
	}

	w = do(r, http.MethodDelete, "/users/alice/secrets/API_KEY?cognito_id=bob", "")
	if w.Code != http.StatusNotFound {
		t.Errorf("DELETE status = %d, want 404", w.Code)
	}
}

func TestUserSecrets_MissingCognitoID(t *testing.T) {
	r := setupRouter(nil)

	w := do(r, http.MethodGet, "/users/alice/secrets", "")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestSystemSecrets_PutNeedsNoPrincipal(t *testing.T) {
	var stored *models.Secret
	store := &mocks.MockSecretStore{
		UpsertSecretFn: func(secret *models.Secret) error {
			stored = secret
			return nil
		},
	}
	r := setupRouter(store)

	w := do(r, http.MethodPut, "/system/secrets",
		`{"name":"NASA_API_KEY","kind":"api_key","value":"nasa-key"}`)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (body: %s)", w.Code, w.Body.String())
	}
	if stored == nil || stored.Scope != models.SystemScope {
		t.Fatalf("stored = %+v, want a system-scoped secret", stored)
	}
	if strings.Contains(w.Body.String(), "nasa-key") {
		t.Error("the response echoed the value")
	}
}

func TestSystemSecrets_ListShowsWhatTheCodeExpects(t *testing.T) {
	store := &mocks.MockSecretStore{
		ListSecretsFn: func(scope models.SecretScope) ([]models.Secret, error) {
			return []models.Secret{{
				Scope:      scope,
				Name:       "EXTRA_KEY",
				Kind:       models.SecretApiKey,
				Ciphertext: []byte("ciphertext-bytes"),
			}}, nil
		},
	}
	r := setupRouter(store)

	w := do(r, http.MethodGet, "/system/secrets", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}

	var statuses []models.SystemSecretStatus
	if err := json.Unmarshal(w.Body.Bytes(), &statuses); err != nil {
		t.Fatalf("decode: %v", err)
	}

	byName := make(map[string]models.SystemSecretStatus)
	for _, s := range statuses {
		byName[s.Name] = s
	}
	if nasa := byName[string(models.SystemNasaApiKey)]; !nasa.Known || nasa.Configured {
		t.Errorf("NASA_API_KEY = %+v, want known and unconfigured", nasa)
	}
	if extra := byName["EXTRA_KEY"]; extra.Known || !extra.Configured {
		t.Errorf("EXTRA_KEY = %+v, want configured and unknown", extra)
	}
	if strings.Contains(w.Body.String(), "ciphertext") {
		t.Error("the listing exposed ciphertext")
	}
}

func TestSystemSecrets_Delete(t *testing.T) {
	var gotScope models.SecretScope
	var gotName string
	store := &mocks.MockSecretStore{
		DeleteSecretFn: func(scope models.SecretScope, name string) error {
			gotScope, gotName = scope, name
			return nil
		},
	}
	r := setupRouter(store)

	w := do(r, http.MethodDelete, "/system/secrets/NASA_API_KEY", "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
	if gotScope != models.SystemScope || gotName != "NASA_API_KEY" {
		t.Errorf("deleted %s/%s", gotScope.Key(), gotName)
	}
}

func TestSystemOAuth2_Put(t *testing.T) {
	var stored *models.Secret
	store := &mocks.MockSecretStore{
		UpsertSecretFn: func(secret *models.Secret) error {
			stored = secret
			return nil
		},
	}
	r := setupRouter(store)

	w := do(r, http.MethodPut, "/system/oauth2",
		`{"name":"PLATFORM_API","flow":"client_credentials","client_id":"id","client_secret":"sec","token_url":"https://p.example/token"}`)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (body: %s)", w.Code, w.Body.String())
	}
	if stored == nil || stored.Scope != models.SystemScope || stored.Kind != models.SecretOAuth2 {
		t.Fatalf("stored = %+v", stored)
	}
}

func TestSystemKeys_ListsMetadataOnly(t *testing.T) {
	store := &mocks.MockSecretStore{}
	r := setupRouter(store)

	// Writing a secret provisions the scope's key.
	if w := do(r, http.MethodPut, "/system/secrets", `{"name":"NASA_API_KEY","kind":"api_key","value":"v"}`); w.Code != http.StatusNoContent {
		t.Fatalf("put status = %d", w.Code)
	}

	w := do(r, http.MethodGet, "/system/keys", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d (body: %s)", w.Code, w.Body.String())
	}

	var keys []models.DataKey
	if err := json.Unmarshal(w.Body.Bytes(), &keys); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(keys) != 1 || keys[0].Scope != models.SystemScope || !keys[0].Active || keys[0].KEKVersion != 1 {
		t.Errorf("keys = %+v", keys)
	}

	body := strings.ToLower(w.Body.String())
	for _, leak := range []string{"wrapped", "nonce"} {
		if strings.Contains(body, leak) {
			t.Errorf("listing exposed %q: %s", leak, body)
		}
	}
}
