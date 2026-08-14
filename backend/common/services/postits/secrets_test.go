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
	"strings"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/ports/crypto"
	"github.com/Secreto31126/tesis/common/ports/executer"
	"github.com/Secreto31126/tesis/common/ports/safehttp"
	secretsrv "github.com/Secreto31126/tesis/common/services/secrets"
	"github.com/google/uuid"
)

// No well-known may name a specific credential: which one to use is the
// user's choice, made per post-it from the board's stored secrets.
func TestWellKnowns_DoNotHardcodeACredential(t *testing.T) {
	for key, wk := range configuredPostIts {
		for _, source := range []map[string]string{wk.Params, wk.Request.Headers, wk.Request.Queries} {
			for field, value := range source {
				name, isRef := strings.CutPrefix(value, "$")
				if isRef && models.ValidSecretName(name) {
					t.Errorf("%s: %q hardcodes the secret %q", key, field, name)
				}
			}
		}
	}
}

// A well-known that needs a credential asks for one, and wires it through a
// param rather than naming a secret itself.
func TestWellKnown_ExchangeRateAsksForACredential(t *testing.T) {
	wk := configuredPostIts["exchange_rate"]

	if wk.Request.Headers["apikey"] != "$credential" {
		t.Errorf("apikey header = %q, want it to defer to the user's choice", wk.Request.Headers["apikey"])
	}
	if def, declared := wk.Params["$credential"]; !declared || def != "" {
		t.Errorf("credential param = %q/%v, want a required param", def, declared)
	}

	// Being required means creation fails until the user picks one.
	if _, err := findWellKnown("exchange_rate", nil); err == nil {
		t.Error("the post-it was created without a credential")
	}
}

// Once the user picks one, it is the picked name that gets resolved.
func TestWellKnown_ExchangeRateUsesTheChosenCredential(t *testing.T) {
	wk, err := findWellKnown("exchange_rate", map[string]string{"$credential": "$MY_CURRENCY_KEY"})
	if err != nil {
		t.Fatalf("findWellKnown: %v", err)
	}

	refs := secretRefs(wk)
	if !slices.Contains(refs, "MY_CURRENCY_KEY") {
		t.Errorf("secretRefs = %v, want the chosen credential", refs)
	}
	for _, unwanted := range []string{"base", "currency", "credential"} {
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
		Params:   map[string]string{"$base": "USD", "$currency": "ARS", "$credential": "$CURRENCY_API_KEY"},
		Request: models.Request{
			Method:  http.MethodGet,
			Queries: map[string]string{"base_currency": "$base", "currencies": "$currency"},
			Headers: map[string]string{"apikey": "$credential"},
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
	if postit.Request.Headers["apikey"] != "$credential" {
		t.Errorf("caller's header became %q", postit.Request.Headers["apikey"])
	}
	if postit.Params["$credential"] != "$CURRENCY_API_KEY" {
		t.Errorf("caller's credential param became %q", postit.Params["$credential"])
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
