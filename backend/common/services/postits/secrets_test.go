package postits

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/ports/crypto"
	"github.com/Secreto31126/tesis/common/ports/executer"
	"github.com/Secreto31126/tesis/common/ports/safehttp"
	secretsrv "github.com/Secreto31126/tesis/common/services/secrets"
	"github.com/google/uuid"
)

// The exchange_rate well-known has to declare the key as a secret reference,
// not as a param, or creating the post-it would demand the key up front.
func TestWellKnown_ExchangeRateReferencesTheSecret(t *testing.T) {
	wk, err := findWellKnown("exchange_rate", nil)
	if err != nil {
		t.Fatalf("findWellKnown: %v", err)
	}

	if wk.Request.Headers["apikey"] != "$CURRENCY_API_KEY" {
		t.Errorf("apikey header = %q", wk.Request.Headers["apikey"])
	}
	if _, isParam := wk.Params["$CURRENCY_API_KEY"]; isParam {
		t.Error("the key is declared as a param, so creation would require it")
	}

	refs := secretRefs(wk)
	if !slices.Contains(refs, "CURRENCY_API_KEY") {
		t.Errorf("secretRefs = %v, want it to include CURRENCY_API_KEY", refs)
	}
	// The ordinary params must not be mistaken for secrets.
	for _, unwanted := range []string{"base", "currency"} {
		if slices.Contains(refs, unwanted) {
			t.Errorf("%q was treated as a secret", unwanted)
		}
	}
}

// End to end: a real sealer, a real secrets service and the real executer,
// with a stand-in for currencyapi that records what it received.
func TestExecutePostIt_InjectsTheStoredApiKey(t *testing.T) {
	original := safehttp.IsSafeIP
	safehttp.IsSafeIP = func(net.IP) bool { return true }
	t.Cleanup(func() { safehttp.IsSafeIP = original })

	var gotKey, gotBase, gotCurrency string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("apikey")
		gotBase = r.URL.Query().Get("base_currency")
		gotCurrency = r.URL.Query().Get("currencies")

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"meta":{"last_updated_at":"2026-08-14T00:00:00Z"},
		                "data":{"ARS":{"code":"ARS","value":1465.5}}}`)
	}))
	defer provider.Close()

	board := uuid.New()
	const owner = "owner-id"
	const theKey = "cur_live_SUPERSECRETVALUE"

	// A real secrets service, so the key is genuinely encrypted and decrypted.
	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x5E}, 32))
	sealer, err := crypto.NewFromKeys("1:" + key)
	if err != nil {
		t.Fatalf("sealer: %v", err)
	}

	var stored []models.Secret
	store := &mocks.MockSecretStore{
		UpsertSecretFn: func(s *models.Secret) error {
			stored = append(stored, *s)
			return nil
		},
		FindSecretsFn: func(_ uuid.UUID, _ []string) ([]models.Secret, error) {
			return stored, nil
		},
	}
	boards := &mocks.MockDB{
		FindBoardFn: func(id uuid.UUID) (*models.Board, error) {
			return &models.Board{Id: id, Owner: owner}, nil
		},
	}

	secrets := secretsrv.New(store, boards, sealer,
		&mocks.MockTokenClient{}, &mocks.MockLocker{}, &mocks.MockHandshakeStore{})

	if err := secrets.Put(board, owner, "CURRENCY_API_KEY", models.SecretApiKey, theKey); err != nil {
		t.Fatalf("put secret: %v", err)
	}
	if bytes.Contains(stored[0].Ciphertext, []byte(theKey)) {
		t.Fatal("the key was not actually encrypted")
	}

	// Same shape as the well-known, pointed at the stand-in.
	resource, _ := url.Parse(provider.URL)
	postit := &models.PostIts{
		Id:       uuid.New(),
		Board:    board,
		Resource: resource,
		Params:   map[string]string{"$base": "USD", "$currency": "ARS"},
		Request: models.Request{
			Method:  http.MethodGet,
			Queries: map[string]string{"base_currency": "$base", "currencies": "$currency"},
			Headers: map[string]string{"apikey": "$CURRENCY_API_KEY"},
		},
		Response: "json",
		Query: map[string]string{
			"code":    ".data[].code",
			"value":   ".data[].value",
			"updated": ".meta.last_updated_at",
		},
	}

	svc := New(&mocks.MockDB{}, &mocks.MockCache{}, executer.New(), secrets)

	data, err := svc.ExecutePostIt(postit)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if gotKey != theKey {
		t.Errorf("provider saw apikey=%q, want the decrypted secret", gotKey)
	}
	if gotBase != "USD" || gotCurrency != "ARS" {
		t.Errorf("provider saw base=%q currencies=%q", gotBase, gotCurrency)
	}

	result, ok := data.(map[string]any)
	if !ok {
		t.Fatalf("result type = %T", data)
	}
	if result["code"] != "ARS" || result["value"] != 1465.5 {
		t.Errorf("result = %v, want ARS/1465.5", result)
	}

	// The key must not have stuck to the post-it the caller still holds.
	if postit.Request.Headers["apikey"] != "$CURRENCY_API_KEY" {
		t.Errorf("caller's header became %q", postit.Request.Headers["apikey"])
	}
	if _, leaked := postit.Params["$CURRENCY_API_KEY"]; leaked {
		t.Error("the decrypted key was written onto the caller's params")
	}
}

// Without the secret configured the reference is sent verbatim, and the
// provider rejects it, rather than the request silently succeeding.
func TestExecutePostIt_MissingSecretIsNotSubstituted(t *testing.T) {
	original := safehttp.IsSafeIP
	safehttp.IsSafeIP = func(net.IP) bool { return true }
	t.Cleanup(func() { safehttp.IsSafeIP = original })

	var gotKey string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("apikey")
		http.Error(w, `{"error":"missing_api_key"}`, http.StatusUnauthorized)
	}))
	defer provider.Close()

	resource, _ := url.Parse(provider.URL)
	postit := &models.PostIts{
		Id:       uuid.New(),
		Board:    uuid.New(),
		Resource: resource,
		Request: models.Request{
			Method:  http.MethodGet,
			Headers: map[string]string{"apikey": "$CURRENCY_API_KEY"},
		},
		Response: "json",
		Query:    map[string]string{"value": ".data[].value"},
	}

	svc := New(&mocks.MockDB{}, &mocks.MockCache{}, executer.New(), &mocks.MockSecretResolver{})

	if _, err := svc.ExecutePostIt(postit); err == nil {
		t.Fatal("expected an error when the provider rejects the credential")
	}
	if gotKey != "$CURRENCY_API_KEY" {
		t.Errorf("provider saw apikey=%q, want the unresolved reference", gotKey)
	}
}
