package secrets

import (
	"encoding/json"
	"github.com/Secreto31126/tesis/common/infrastructure"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

// rememberingStore keeps rows so a handler can read back what another wrote.
func rememberingStore() *mocks.MockSecretStore {
	var rows []models.Secret
	return &mocks.MockSecretStore{
		UpsertSecretFn: func(secret *models.Secret) error {
			rows = slices.DeleteFunc(rows, func(row models.Secret) bool {
				return row.Scope == secret.Scope && row.Name == secret.Name
			})
			rows = append(rows, *secret)
			return nil
		},
		FindSecretsFn: func(scope models.SecretScope, names []string) ([]models.Secret, error) {
			var out []models.Secret
			for _, row := range rows {
				if row.Scope == scope && slices.Contains(names, row.Name) {
					out = append(out, row)
				}
			}
			return out, nil
		},
		ListSecretsFn: func(scope models.SecretScope) ([]models.Secret, error) {
			var out []models.Secret
			for _, row := range rows {
				if row.Scope == scope {
					out = append(out, row)
				}
			}
			return out, nil
		},
		SetGrantsFn: func(scope models.SecretScope, name string, grants []models.Grant) error {
			for i, row := range rows {
				if row.Scope == scope && row.Name == name {
					rows[i].Grants = grants
					return nil
				}
			}
			return infrastructure.ErrUnknownCredential
		},
		FindGrantedFn: func(audiences []models.Audience) ([]models.Secret, error) {
			var out []models.Secret
			for _, row := range rows {
				for _, grant := range row.Grants {
					if slices.Contains(audiences, grant.To) {
						out = append(out, row)
						break
					}
				}
			}
			return out, nil
		},
	}
}

func TestGrants_OwnerSharesAndTheListingShowsIt(t *testing.T) {
	store := rememberingStore()
	r := setupRouter(store)
	board := uuid.New()
	path := "/users/alice/secrets"

	w := do(r, http.MethodPut, path, `{"cognito_id":"alice","name":"KEY","kind":"api_key","value":"v"}`)
	if w.Code != http.StatusNoContent {
		t.Fatalf("PUT: status = %d (body: %s)", w.Code, w.Body.String())
	}

	body := `{"cognito_id":"alice","grants":[{"to":{"kind":"user","id":"` + collaborator + `"},"board":"` + board.String() + `"}]}`
	w = do(r, http.MethodPut, path+"/KEY/grants", body)
	if w.Code != http.StatusNoContent {
		t.Fatalf("PUT grants: status = %d (body: %s)", w.Code, w.Body.String())
	}

	var metas []models.SecretMeta
	w = do(r, http.MethodGet, path+"?cognito_id=alice", "")
	if err := json.Unmarshal(w.Body.Bytes(), &metas); err != nil || len(metas) != 1 {
		t.Fatalf("list: status = %d, body = %s", w.Code, w.Body.String())
	}
	if len(metas[0].Grants) != 1 || metas[0].Grants[0].To.ID != collaborator || metas[0].Grants[0].Board != board.String() {
		t.Errorf("grants in the listing = %+v", metas[0].Grants)
	}

	w = do(r, http.MethodGet, "/boards/"+board.String()+"/secrets/usable?cognito_id="+collaborator, "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"KEY"`) {
		t.Errorf("usable for the grantee: status = %d, body = %s", w.Code, w.Body.String())
	}
	w = do(r, http.MethodGet, "/boards/"+uuid.New().String()+"/secrets/usable?cognito_id="+collaborator, "")
	if strings.Contains(w.Body.String(), `"KEY"`) {
		t.Errorf("the grant leaked onto another board: %s", w.Body.String())
	}

	if w = do(r, http.MethodPut, path+"/KEY/grants", `{"cognito_id":"bob","grants":[]}`); w.Code != http.StatusNotFound {
		t.Errorf("someone else changed the grants: status = %d, want 404", w.Code)
	}
	if w = do(r, http.MethodPut, path+"/NOPE/grants", `{"cognito_id":"alice","grants":[]}`); w.Code != http.StatusBadRequest {
		t.Errorf("granting a missing secret: status = %d, want 400", w.Code)
	}
	if w = do(r, http.MethodPut, path+"/KEY/grants", `{"cognito_id":"alice","grants":[{"to":{"kind":"org","id":"x"}}]}`); w.Code != http.StatusBadRequest {
		t.Errorf("an invalid grant: status = %d, want 400", w.Code)
	}
	if w = do(r, http.MethodPut, "/system/secrets/KEY/grants", `{"grants":[]}`); w.Code != http.StatusNotFound {
		t.Errorf("system secrets have a grants route: status = %d, want 404", w.Code)
	}
}

func TestMemberSecrets_OnlyThatMemberSeesThem(t *testing.T) {
	store := rememberingStore()
	r := setupRouter(store)
	board := uuid.New()
	path := "/boards/" + board.String() + "/members/" + collaborator + "/secrets"

	w := do(r, http.MethodPut, path, `{"cognito_id":"`+collaborator+`","name":"MINE","kind":"api_key","value":"v"}`)
	if w.Code != http.StatusNoContent {
		t.Fatalf("PUT status = %d, want 204 (body: %s)", w.Code, w.Body.String())
	}

	w = do(r, http.MethodGet, path+"?cognito_id="+collaborator, "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"MINE"`) {
		t.Errorf("GET by the member: status = %d, body = %s", w.Code, w.Body.String())
	}

	for _, caller := range []string{owner, "stranger"} {
		w = do(r, http.MethodGet, path+"?cognito_id="+caller, "")
		if w.Code != http.StatusNotFound {
			t.Errorf("GET by %s: status = %d, want 404", caller, w.Code)
		}
		w = do(r, http.MethodPut, path, `{"cognito_id":"`+caller+`","name":"MINE","kind":"api_key","value":"x"}`)
		if w.Code != http.StatusNotFound {
			t.Errorf("PUT by %s: status = %d, want 404", caller, w.Code)
		}
	}

	w = do(r, http.MethodPut, "/boards/"+board.String()+"/members/stranger/secrets", `{"cognito_id":"stranger","name":"MINE","kind":"api_key","value":"x"}`)
	if w.Code != http.StatusNotFound {
		t.Errorf("a non-member got a member scope: status = %d, want 404", w.Code)
	}
}

func TestUsableSecrets_ListsWhatTheCallerCanBind(t *testing.T) {
	store := rememberingStore()
	r := setupRouter(store)
	board := uuid.New()

	put := func(path, caller, name string) {
		t.Helper()
		w := do(r, http.MethodPut, path, `{"cognito_id":"`+caller+`","name":"`+name+`","kind":"api_key","value":"v"}`)
		if w.Code != http.StatusNoContent {
			t.Fatalf("PUT %s: status = %d (body: %s)", path, w.Code, w.Body.String())
		}
	}
	put("/boards/"+board.String()+"/secrets", owner, "SHARED")
	put("/boards/"+board.String()+"/members/"+collaborator+"/secrets", collaborator, "MINE")
	put("/boards/"+board.String()+"/members/"+owner+"/secrets", owner, "OWNERS")
	put("/users/"+collaborator+"/secrets", collaborator, "PROFILE")

	w := do(r, http.MethodGet, "/boards/"+board.String()+"/secrets/usable?cognito_id="+collaborator, "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
	}

	var usable []models.SecretMeta
	if err := json.Unmarshal(w.Body.Bytes(), &usable); err != nil {
		t.Fatalf("decode: %v", err)
	}
	got := map[string]models.ScopeKind{}
	for _, meta := range usable {
		got[meta.Name] = meta.Scope.Kind
	}
	want := map[string]models.ScopeKind{"SHARED": models.ScopeBoard, "MINE": models.ScopeMember, "PROFILE": models.ScopeUser}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for name, kind := range want {
		if got[name] != kind {
			t.Errorf("%s: kind %q, want %q", name, got[name], kind)
		}
	}
	if strings.Contains(w.Body.String(), `"v"`) || strings.Contains(w.Body.String(), "ciphertext") {
		t.Error("the listing carries secret material")
	}

	w = do(r, http.MethodGet, "/boards/"+board.String()+"/secrets/usable?cognito_id=stranger", "")
	if w.Code != http.StatusNotFound {
		t.Errorf("stranger: status = %d, want 404", w.Code)
	}
}

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

func withPlatformGoogle(t *testing.T, r http.Handler) {
	t.Helper()
	w := do(r, http.MethodPut, "/system/oauth2/clients", `{"provider":"google","client_id":"pid","client_secret":"psecret"}`)
	if w.Code != http.StatusNoContent {
		t.Fatalf("put client status = %d (body: %s)", w.Code, w.Body.String())
	}
}

func TestPlatformClient_ResponseCarriesNothing(t *testing.T) {
	r := setupRouter(rememberingStore())

	w := do(r, http.MethodPut, "/system/oauth2/clients", `{"provider":"google","client_id":"pid","client_secret":"psecret"}`)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d (body: %s)", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "psecret") {
		t.Error("the response echoed the client secret")
	}

	w = do(r, http.MethodPut, "/system/oauth2/clients", `{"provider":"myspace","client_id":"pid","client_secret":"psecret"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("unknown provider status = %d, want 400", w.Code)
	}
}

func TestConnect_BoardAndUserScopes(t *testing.T) {
	r := setupRouter(rememberingStore())
	withPlatformGoogle(t, r)
	board := uuid.New()

	for _, path := range []string{"/boards/" + board.String() + "/oauth2/connect", "/users/" + owner + "/oauth2/connect"} {
		w := do(r, http.MethodPost, path,
			`{"cognito_id":"`+owner+`","provider":"google","name":"GOOGLE","redirect_uri":"https://tablero.example/oauth2/callback"}`)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: status = %d (body: %s)", path, w.Code, w.Body.String())
		}

		var body map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if !strings.HasPrefix(body["authorization_url"], "https://accounts.google.com/") {
			t.Errorf("%s: authorization_url = %q", path, body["authorization_url"])
		}
		if len(body) != 1 {
			t.Errorf("%s: response carries more than the url: %v", path, body)
		}
		if strings.Contains(w.Body.String(), "psecret") {
			t.Error("the client secret went to the client")
		}
	}
}

func TestConnect_StrangerIsNotFound(t *testing.T) {
	r := setupRouter(rememberingStore())
	withPlatformGoogle(t, r)

	w := do(r, http.MethodPost, "/boards/"+uuid.New().String()+"/oauth2/connect",
		`{"cognito_id":"stranger","provider":"google","name":"GOOGLE","redirect_uri":"https://tablero.example/oauth2/callback"}`)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestConnect_UnprovisionedProviderIsServiceUnavailable(t *testing.T) {
	r := setupRouter(rememberingStore())

	w := do(r, http.MethodPost, "/boards/"+uuid.New().String()+"/oauth2/connect",
		`{"cognito_id":"`+owner+`","provider":"google","name":"GOOGLE","redirect_uri":"https://tablero.example/oauth2/callback"}`)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503 (body: %s)", w.Code, w.Body.String())
	}
}

func TestProviders_ListsConfiguredFlag(t *testing.T) {
	r := setupRouter(rememberingStore())
	withPlatformGoogle(t, r)

	w := do(r, http.MethodGet, "/oauth2/providers", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var statuses []models.OAuthProviderStatus
	_ = json.Unmarshal(w.Body.Bytes(), &statuses)
	byName := map[models.OAuthProvider]bool{}
	for _, s := range statuses {
		byName[s.Provider] = s.Configured
	}
	if !byName[models.ProviderGoogle] || byName[models.ProviderDiscord] {
		t.Errorf("statuses = %+v", statuses)
	}
}

func TestGroupSecrets_OwnerManagesMembersView(t *testing.T) {
	store := rememberingStore()
	r := setupRouter(store)
	path := "/groups/" + opsGroup.Id.String() + "/secrets"

	w := do(r, http.MethodPut, path, `{"cognito_id":"`+owner+`","name":"OPS_KEY","kind":"api_key","value":"v"}`)
	if w.Code != http.StatusNoContent {
		t.Fatalf("PUT by the owner: status = %d (body: %s)", w.Code, w.Body.String())
	}

	w = do(r, http.MethodPut, path, `{"cognito_id":"`+collaborator+`","name":"OPS_KEY","kind":"api_key","value":"x"}`)
	if w.Code != http.StatusNotFound {
		t.Errorf("PUT by a member: status = %d, want 404", w.Code)
	}

	for _, caller := range []string{owner, collaborator} {
		w = do(r, http.MethodGet, path+"?cognito_id="+caller, "")
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"OPS_KEY"`) {
			t.Errorf("GET by %s: status = %d, body = %s", caller, w.Code, w.Body.String())
		}
	}
	if w = do(r, http.MethodGet, path+"?cognito_id=stranger", ""); w.Code != http.StatusNotFound {
		t.Errorf("GET by a stranger: status = %d, want 404", w.Code)
	}

	w = do(r, http.MethodGet, "/boards/"+uuid.New().String()+"/secrets/usable?cognito_id="+collaborator, "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"OPS_KEY"`) {
		t.Errorf("usable for a member: status = %d, body = %s", w.Code, w.Body.String())
	}
}
