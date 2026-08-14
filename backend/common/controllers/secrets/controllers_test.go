package secrets

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/ports/crypto"
	srv "github.com/Secreto31126/tesis/common/services/secrets"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const owner = "owner-id"

func setupRouter(store *mocks.MockSecretStore) *gin.Engine {
	if store == nil {
		store = &mocks.MockSecretStore{}
	}

	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x7C}, 32))
	sealer, _ := crypto.NewFromKeys("1:" + key)

	boards := &mocks.MockDB{
		FindBoardFn: func(id uuid.UUID) (*models.Board, error) {
			return &models.Board{Id: id, Owner: owner}, nil
		},
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewController(srv.New(store, boards, sealer)).RegisterRoutes(r)
	return r
}

func do(r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestPutSecret_OK(t *testing.T) {
	var stored *models.Secret
	store := &mocks.MockSecretStore{
		UpsertSecretFn: func(secret *models.Secret) error {
			stored = secret
			return nil
		},
	}
	r := setupRouter(store)
	board := uuid.New()

	w := do(r, http.MethodPut, "/boards/"+board.String()+"/secrets",
		`{"cognito_id":"`+owner+`","name":"API_KEY","kind":"api_key","value":"s3cr3t"}`)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (body: %s)", w.Code, w.Body.String())
	}
	if stored == nil || stored.Name != "API_KEY" {
		t.Fatalf("stored = %+v", stored)
	}
	if bytes.Contains(stored.Ciphertext, []byte("s3cr3t")) {
		t.Error("the value was persisted in the clear")
	}
}

// Nothing this API returns may contain the value the caller just wrote.
func TestPutSecret_ResponseCarriesNoValue(t *testing.T) {
	r := setupRouter(nil)
	board := uuid.New()

	w := do(r, http.MethodPut, "/boards/"+board.String()+"/secrets",
		`{"cognito_id":"`+owner+`","name":"API_KEY","kind":"api_key","value":"s3cr3t"}`)

	if strings.Contains(w.Body.String(), "s3cr3t") {
		t.Errorf("the response echoed the secret: %s", w.Body.String())
	}
}

func TestPutSecret_RejectsNonOwnerAsNotFound(t *testing.T) {
	written := false
	store := &mocks.MockSecretStore{
		UpsertSecretFn: func(_ *models.Secret) error {
			written = true
			return nil
		},
	}
	r := setupRouter(store)
	board := uuid.New()

	w := do(r, http.MethodPut, "/boards/"+board.String()+"/secrets",
		`{"cognito_id":"stranger","name":"API_KEY","kind":"api_key","value":"v"}`)

	// 404 rather than 403: a stranger should not learn the board exists.
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
	if written {
		t.Error("a stranger's write reached the store")
	}
}

func TestPutSecret_RejectsBadName(t *testing.T) {
	r := setupRouter(nil)
	board := uuid.New()

	w := do(r, http.MethodPut, "/boards/"+board.String()+"/secrets",
		`{"cognito_id":"`+owner+`","name":"lower_case","kind":"api_key","value":"v"}`)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (body: %s)", w.Code, w.Body.String())
	}
}

func TestPutSecret_InvalidBoard(t *testing.T) {
	r := setupRouter(nil)
	w := do(r, http.MethodPut, "/boards/not-a-uuid/secrets",
		`{"cognito_id":"`+owner+`","name":"API_KEY","kind":"api_key","value":"v"}`)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestListSecrets_ReturnsMetadataOnly(t *testing.T) {
	board := uuid.New()
	store := &mocks.MockSecretStore{
		ListSecretsFn: func(_ uuid.UUID) ([]models.Secret, error) {
			return []models.Secret{{
				Board:      board,
				Name:       "API_KEY",
				Kind:       models.SecretApiKey,
				Ciphertext: []byte("ciphertext-bytes"),
				Nonce:      []byte("nonce-bytes"),
			}}, nil
		},
	}
	r := setupRouter(store)

	w := do(r, http.MethodGet, "/boards/"+board.String()+"/secrets?cognito_id="+owner, "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "API_KEY") {
		t.Errorf("expected the name in the listing: %s", body)
	}
	for _, leak := range []string{"ciphertext", "nonce", "keyversion"} {
		if strings.Contains(strings.ToLower(body), leak) {
			t.Errorf("listing exposed %q: %s", leak, body)
		}
	}
}

func TestListSecrets_MissingCognitoID(t *testing.T) {
	r := setupRouter(nil)
	w := do(r, http.MethodGet, "/boards/"+uuid.New().String()+"/secrets", "")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestDeleteSecret_OK(t *testing.T) {
	var gotName string
	store := &mocks.MockSecretStore{
		DeleteSecretFn: func(_ uuid.UUID, name string) error {
			gotName = name
			return nil
		},
	}
	r := setupRouter(store)

	w := do(r, http.MethodDelete, "/boards/"+uuid.New().String()+"/secrets/API_KEY?cognito_id="+owner, "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (body: %s)", w.Code, w.Body.String())
	}
	if gotName != "API_KEY" {
		t.Errorf("deleted %q, want API_KEY", gotName)
	}
}

func TestDeleteSecret_NonOwner(t *testing.T) {
	deleted := false
	store := &mocks.MockSecretStore{
		DeleteSecretFn: func(_ uuid.UUID, _ string) error {
			deleted = true
			return nil
		},
	}
	r := setupRouter(store)

	w := do(r, http.MethodDelete, "/boards/"+uuid.New().String()+"/secrets/API_KEY?cognito_id=stranger", "")
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
	if deleted {
		t.Error("a stranger deleted a secret")
	}
}
