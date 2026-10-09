//go:build integration

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	a_ctrl "github.com/Secreto31126/tesis/common/controllers/auth"
	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/ports/crypto"
	"github.com/Secreto31126/tesis/common/ports/jwt"
	"github.com/Secreto31126/tesis/common/ports/mongo"
	"github.com/Secreto31126/tesis/common/ports/oidc"
	"github.com/Secreto31126/tesis/common/ports/redis"
	"github.com/Secreto31126/tesis/common/ports/safehttp"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	nasaKey = "nasa_live_ONLY_THE_PROVIDER_MAY_SEE_THIS"
	// rootEmail is a platform admin (see newStack) that no test registers by
	// hand, so any test may sign in as it.
	rootEmail  = "root@it.test"
	itPassword = "integration password"
)

// stack wires the real application over the Mongo and Redis named by the
// environment (see run_backend_integration.*) and records every response so
// the test can prove none of them carried a secret.
type stack struct {
	t         *testing.T
	app       *app
	db        *mongo.MongoDB
	cache     *redis.RedisDB
	mailer    *mocks.RecordingMailer
	responses []*httptest.ResponseRecorder
}

func newStack(t *testing.T) *stack {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := mongo.New()
	if err != nil {
		t.Fatalf("mongo: %v", err)
	}
	if err := db.EnsureIndexes(); err != nil {
		t.Skipf("mongo not reachable: %v", err)
	}

	cache, err := redis.New()
	if err != nil {
		t.Fatalf("redis: %v", err)
	}

	kek, err := crypto.NewFromKeys("1:" + base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x11}, 32)))
	if err != nil {
		t.Fatalf("kek: %v", err)
	}

	keys, err := jwt.NewFromKeys("it:" + base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x22}, 32)))
	if err != nil {
		t.Fatalf("signing keys: %v", err)
	}

	t.Cleanup(func() {
		_ = cache.Close()
		_ = db.Close()
	})

	cfg := config{admins: []string{"admin@it.test", rootEmail}, accessTTL: DEFAULT_ACCESS_TTL, refreshTTL: DEFAULT_REFRESH_TTL, cookieSecure: true,
		google: oidc.Config{Issuer: googleIssuer, ClientID: googleClientID, ClientSecret: googleClientSecret}}
	mailer := &mocks.RecordingMailer{}

	return &stack{t: t, app: newApp(db, cache, kek, keys, cfg, mailer), db: db, cache: cache, mailer: mailer}
}

func (s *stack) request(token, method, path string, body any) *httptest.ResponseRecorder {
	s.t.Helper()
	return s.requestFrom("", token, method, path, body)
}

// requestFrom sends the request from the given client address (host:port),
// or httptest's default one when it is empty.
func (s *stack) requestFrom(remote, token, method, path string, body any) *httptest.ResponseRecorder {
	s.t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, _ := json.Marshal(body)
		reader = bytes.NewReader(encoded)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if remote != "" {
		req.RemoteAddr = remote
	}
	w := httptest.NewRecorder()
	s.app.router.ServeHTTP(w, req)
	s.responses = append(s.responses, w)
	return w
}

// do calls the API anonymously.
func (s *stack) do(method, path string, body any) *httptest.ResponseRecorder {
	s.t.Helper()
	return s.request("", method, path, body)
}

func (s *stack) mustJSON(w *httptest.ResponseRecorder, want int, out any) {
	s.t.Helper()
	if w.Code != want {
		s.t.Fatalf("status = %d, want %d (body: %s)", w.Code, want, w.Body.String())
	}
	if out != nil {
		if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
			s.t.Fatalf("decode %s: %v", w.Body.String(), err)
		}
	}
}

func (s *stack) assertNoResponseContains(needle string) {
	s.t.Helper()
	for i, w := range s.responses {
		if strings.Contains(w.Body.String(), needle) {
			s.t.Errorf("response #%d carried %q: %s", i, needle, w.Body.String())
		}
	}
}

// client calls the API as one signed-in user.
type client struct {
	s     *stack
	id    string
	email string
	token string
}

func (c *client) do(method, path string, body any) *httptest.ResponseRecorder {
	c.s.t.Helper()
	return c.s.request(c.token, method, path, body)
}

// signUp registers and verifies a fresh account; who only makes the address
// readable, every call gets a new one.
func (s *stack) signUp(who string) *client {
	s.t.Helper()
	return s.register(who + "-" + uuid.NewString()[:8] + "@it.test")
}

// freshAddress is a client address nobody used yet. Registering and signing
// in are limited per client, and the suite signs in far more users than one
// client may.
func freshAddress() string {
	id := uuid.New()
	return net.IPv4(10, id[0], id[1], id[2]).String() + ":1234"
}

func (s *stack) register(email string) *client {
	s.t.Helper()
	s.mustJSON(s.requestFrom(freshAddress(), "", http.MethodPost, "/api/v1/auth/register", map[string]string{"email": email, "password": itPassword, "name": email}), http.StatusAccepted, nil)

	var session a_ctrl.SessionResponse
	s.mustJSON(s.do(http.MethodPost, "/api/v1/auth/verify-email", map[string]string{"token": s.lastMailedToken()}), http.StatusOK, &session)
	return &client{s, session.User.Id.String(), email, session.AccessToken}
}

// admin signs in as the seeded platform admin, registering it on first use.
func (s *stack) admin() *client {
	s.t.Helper()
	var session a_ctrl.SessionResponse
	w := s.requestFrom(freshAddress(), "", http.MethodPost, "/api/v1/auth/login", map[string]string{"email": rootEmail, "password": itPassword})
	if w.Code != http.StatusOK {
		return s.register(rootEmail)
	}
	s.mustJSON(w, http.StatusOK, &session)
	return &client{s, session.User.Id.String(), rootEmail, session.AccessToken}
}

// share puts who on the board with role, as the board's owner c.
func (c *client) share(board models.Board, who *client, role models.BoardRole) {
	c.s.t.Helper()
	c.s.mustJSON(c.do(http.MethodPut, "/api/v1/boards/"+board.Id.String()+"/members",
		map[string]any{"email": who.email, "role": role}), http.StatusOK, nil)
}

func (c *client) newBoard(name string) models.Board {
	c.s.t.Helper()
	var board models.Board
	c.s.mustJSON(c.do(http.MethodPost, "/api/v1/boards", map[string]string{"name": name}), http.StatusCreated, &board)
	return board
}

// Everything but /auth needs a token: the wiring is the only thing that puts
// a route behind RequireAuth, so walk what is actually mounted.
func TestEndToEnd_EveryPrivateRouteNeedsAToken(t *testing.T) {
	s := newStack(t)
	params := strings.NewReplacer(":id", uuid.NewString(), ":strand", uuid.NewString(), ":user", "u", ":name", "KEY")

	checked := 0
	for _, route := range s.app.router.Routes() {
		if strings.HasPrefix(route.Path, "/api/v1/auth/") {
			continue
		}
		checked++
		if w := s.do(route.Method, params.Replace(route.Path), nil); w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s = %d, want 401", route.Method, route.Path, w.Code)
		}
	}
	if checked < 30 {
		t.Errorf("only %d private routes were found; is the router wired?", checked)
	}
}

// Someone outside a board cannot read, draw on, run or move anything on it,
// and cannot tell it from a board that does not exist.
func TestEndToEnd_AStrangerSeesNothingOfABoard(t *testing.T) {
	s := newStack(t)
	owner, ana, eve := s.signUp("owner"), s.signUp("ana"), s.signUp("eve")

	board := owner.newBoard("it")
	defer owner.do(http.MethodDelete, "/api/v1/boards/"+board.Id.String(), nil)
	base := "/api/v1/boards/" + board.Id.String()
	owner.share(board, ana, models.BoardEditor)

	var postit models.PostIts
	s.mustJSON(ana.do(http.MethodPost, "/api/v1/post-its", map[string]any{"board": board.Id, "well_known": "dolar_oficial"}), http.StatusCreated, &postit)
	card := "/api/v1/post-its/" + postit.Id.String()

	for _, c := range []struct{ method, path string }{
		{http.MethodGet, base},
		{http.MethodGet, base + "/post-its"},
		{http.MethodPatch, base + "/name"},
		{http.MethodDelete, base},
		{http.MethodGet, base + "/secrets"},
		{http.MethodGet, card + "/settings"},
		{http.MethodGet, card},
		{http.MethodPatch, card + "/position"},
		{http.MethodDelete, card},
	} {
		body := map[string]any{"name": "mine now", "x": 1, "y": 2}
		if w := eve.do(c.method, c.path, body); w.Code != http.StatusNotFound {
			t.Errorf("stranger %s %s = %d, want 404 (body: %s)", c.method, c.path, w.Code, w.Body.String())
		}
	}

	// The editor can work on it but not rename it; the board is in both
	// their listings and not in the stranger's.
	if w := ana.do(http.MethodPatch, base+"/name", map[string]string{"name": "mine now"}); w.Code != http.StatusNotFound {
		t.Errorf("an editor renamed the board: %d", w.Code)
	}
	for who, c := range map[string]*client{"owner": owner, "ana": ana, "eve": eve} {
		var boards []models.Board
		s.mustJSON(c.do(http.MethodGet, "/api/v1/boards", nil), http.StatusOK, &boards)
		listed := len(boards) == 1 && boards[0].Id == board.Id
		if listed == (who == "eve") {
			t.Errorf("%s's boards = %+v", who, boards)
		}
	}
}

// The operator routes belong to platform admins.
func TestEndToEnd_SystemRoutesAreForAdmins(t *testing.T) {
	s := newStack(t)
	user, admin := s.signUp("user"), s.admin()

	if w := user.do(http.MethodGet, "/api/v1/system/secrets", nil); w.Code != http.StatusForbidden {
		t.Errorf("a user listed the system secrets: %d", w.Code)
	}
	if w := user.do(http.MethodPut, "/api/v1/system/secrets",
		map[string]string{"name": "EXTRA_KEY", "kind": "api_key", "value": "v"}); w.Code != http.StatusForbidden {
		t.Errorf("a user wrote a system secret: %d", w.Code)
	}
	s.mustJSON(admin.do(http.MethodGet, "/api/v1/system/secrets", nil), http.StatusOK, nil)
	s.mustJSON(admin.do(http.MethodGet, "/api/v1/system/keys", nil), http.StatusOK, nil)
}

func TestEndToEnd_SystemSecretReachesTheProviderAndNoClient(t *testing.T) {
	original := safehttp.IsSafeIP
	safehttp.IsSafeIP = func(net.IP) bool { return true }
	t.Cleanup(func() { safehttp.IsSafeIP = original })

	var seenKey string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenKey = r.URL.Query().Get("api_key")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"title":"Pillars of Creation","url":"https://apod.nasa.gov/p.jpg","explanation":"Gas.","date":"2026-09-19"}`)
	}))
	defer provider.Close()

	s := newStack(t)
	admin, user := s.admin(), s.signUp("user")

	// The platform provisions its key; the listing shows it is there.
	s.mustJSON(admin.do(http.MethodPut, "/api/v1/system/secrets",
		map[string]string{"name": string(models.SystemNasaApiKey), "kind": "api_key", "value": nasaKey}), http.StatusNoContent, nil)
	defer admin.do(http.MethodDelete, "/api/v1/system/secrets/"+string(models.SystemNasaApiKey), nil)

	var statuses []models.SystemSecretStatus
	s.mustJSON(admin.do(http.MethodGet, "/api/v1/system/secrets", nil), http.StatusOK, &statuses)
	configured := false
	for _, st := range statuses {
		configured = configured || (st.Name == string(models.SystemNasaApiKey) && st.Configured)
	}
	if !configured {
		t.Fatalf("NASA_API_KEY not reported configured: %+v", statuses)
	}

	// A user builds a board and drops the card on it.
	board := user.newBoard("it")

	var postit models.PostIts
	s.mustJSON(user.do(http.MethodPost, "/api/v1/post-its",
		map[string]any{"board": board.Id, "well_known": "nasa_apod"}), http.StatusCreated, &postit)

	if postit.Request.Queries["api_key"] != "$api_key" {
		t.Fatalf("stored query = %q, want the placeholder", postit.Request.Queries["api_key"])
	}

	// Point the persisted card at the stub, the way it would be read back from
	// the database, without touching the request template.
	stub, _ := url.Parse(provider.URL)
	if err := s.db.UpdatePostIt(postit.Id, map[string]any{"resource": stub}); err != nil {
		t.Fatalf("retarget: %v", err)
	}

	var settings models.PostIts
	settingsResponse := user.do(http.MethodGet, "/api/v1/post-its/"+postit.Id.String()+"/settings", nil)
	s.mustJSON(settingsResponse, http.StatusOK, &settings)

	var result map[string]any
	executionResponse := user.do(http.MethodGet, "/api/v1/post-its/"+postit.Id.String(), nil)
	s.mustJSON(executionResponse, http.StatusOK, &result)

	if seenKey != nasaKey {
		t.Errorf("provider saw api_key=%q, want the system secret", seenKey)
	}
	if result["title"] != "Pillars of Creation" {
		t.Errorf("result = %v", result)
	}

	// Executing again is served from Redis; the provider is not called twice.
	seenKey = ""
	s.mustJSON(user.do(http.MethodGet, "/api/v1/post-its/"+postit.Id.String(), nil), http.StatusOK, &result)
	if seenKey != "" {
		t.Error("the second execution hit the provider instead of the cache")
	}

	// Deleting the board purges the scope, and the platform's key stays.
	s.mustJSON(user.do(http.MethodDelete, "/api/v1/boards/"+board.Id.String(), nil), http.StatusNoContent, nil)
	if _, err := s.db.FindActiveKey(models.BoardScope(board.Id)); err == nil {
		t.Error("the board's data key survived the delete")
	}
	if _, err := s.db.FindActiveKey(models.SystemScope); err != nil {
		t.Errorf("the system data key is gone: %v", err)
	}

	// The value never leaves the server. The name only appears in the operator
	// listing under /system; what a board user gets back never mentions it.
	s.assertNoResponseContains(nasaKey)
	for _, w := range []*httptest.ResponseRecorder{settingsResponse, executionResponse} {
		if strings.Contains(w.Body.String(), string(models.SystemNasaApiKey)) {
			t.Errorf("a user-facing response names the system secret: %s", w.Body.String())
		}
	}
}

// Without the platform key the card is unavailable, and the failure does not
// say which credential is missing.
func TestEndToEnd_MissingSystemSecretIsUnavailable(t *testing.T) {
	s := newStack(t)
	user := s.signUp("user")
	s.admin().do(http.MethodDelete, "/api/v1/system/secrets/"+string(models.SystemNasaApiKey), nil)

	board := user.newBoard("it")
	defer user.do(http.MethodDelete, "/api/v1/boards/"+board.Id.String(), nil)

	var postit models.PostIts
	s.mustJSON(user.do(http.MethodPost, "/api/v1/post-its",
		map[string]any{"board": board.Id, "well_known": "nasa_apod"}), http.StatusCreated, &postit)

	w := user.do(http.MethodGet, "/api/v1/post-its/"+postit.Id.String(), nil)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (body: %s)", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "NASA") {
		t.Errorf("the error names the secret: %s", w.Body.String())
	}
}

// A board's owner can list what exists but never read a value back, and a
// stranger cannot even learn the board exists.
func TestEndToEnd_BoardSecretsAreWriteOnly(t *testing.T) {
	s := newStack(t)
	owner, stranger := s.signUp("owner"), s.signUp("stranger")

	board := owner.newBoard("it")
	defer owner.do(http.MethodDelete, "/api/v1/boards/"+board.Id.String(), nil)
	base := "/api/v1/boards/" + board.Id.String()

	s.mustJSON(owner.do(http.MethodPut, base+"/secrets",
		map[string]string{"name": "MY_KEY", "kind": "bearer", "value": "board-secret-value"}), http.StatusNoContent, nil)

	var metas []models.SecretMeta
	s.mustJSON(owner.do(http.MethodGet, base+"/secrets", nil), http.StatusOK, &metas)
	if len(metas) != 1 || metas[0].Name != "MY_KEY" || metas[0].Scope != models.BoardScope(board.Id) {
		t.Errorf("metas = %+v", metas)
	}

	if w := stranger.do(http.MethodGet, base+"/secrets", nil); w.Code != http.StatusNotFound {
		t.Errorf("stranger got %d, want 404", w.Code)
	}

	s.assertNoResponseContains("board-secret-value")
}

// A credential a member keeps for themselves on one board: the other members
// cannot list it, cannot bind it, and it dies with the membership.
func TestEndToEnd_MemberSecretIsOnlyBindableByItsOwner(t *testing.T) {
	original := safehttp.IsSafeIP
	safehttp.IsSafeIP = func(net.IP) bool { return true }
	t.Cleanup(func() { safehttp.IsSafeIP = original })

	const (
		anasKey = "cur_live_ONLY_ANAS_CARD_MAY_SEND_THIS"
		keyName = "ANAS_CURRENCY_KEY"
	)

	var seenKeys []string
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenKeys = append(seenKeys, r.Header.Get("apikey"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"meta":{"last_updated_at":"2026-09-20T00:00:00Z"},"data":{"ARS":{"code":"ARS","value":1465.5}}}`)
	}))
	defer stub.Close()

	s := newStack(t)
	owner, ana := s.signUp("owner"), s.signUp("ana")

	board := owner.newBoard("it")
	defer owner.do(http.MethodDelete, "/api/v1/boards/"+board.Id.String(), nil)
	base := "/api/v1/boards/" + board.Id.String()
	owner.share(board, ana, models.BoardEditor)

	// Ana keeps her key on this board, for herself.
	mine := base + "/members/" + ana.id + "/secrets"
	s.mustJSON(ana.do(http.MethodPut, mine,
		map[string]string{"name": keyName, "kind": "api_key", "value": anasKey}), http.StatusNoContent, nil)

	if w := owner.do(http.MethodGet, mine, nil); w.Code != http.StatusNotFound {
		t.Errorf("the board owner listed Ana's secrets: %d", w.Code)
	}

	var usable []models.SecretMeta
	s.mustJSON(ana.do(http.MethodGet, base+"/secrets/usable", nil), http.StatusOK, &usable)
	if len(usable) != 1 || usable[0].Name != keyName || usable[0].Scope != models.MemberScope(board.Id, ana.id) {
		t.Errorf("Ana's usable secrets = %+v", usable)
	}
	s.mustJSON(owner.do(http.MethodGet, base+"/secrets/usable", nil), http.StatusOK, &usable)
	if len(usable) != 0 {
		t.Errorf("the owner's usable secrets include Ana's: %+v", usable)
	}

	card := map[string]any{
		"board":      board.Id,
		"well_known": "exchange_rate",
		"params":     map[string]string{"$credential": "$" + keyName},
		"bindings":   map[string]any{keyName: map[string]any{"scope": models.MemberScope(board.Id, ana.id), "name": keyName}},
	}

	if w := owner.do(http.MethodPost, "/api/v1/post-its", card); w.Code != http.StatusForbidden {
		t.Errorf("the owner bound Ana's secret: %d %s", w.Code, w.Body.String())
	}

	var postit models.PostIts
	s.mustJSON(ana.do(http.MethodPost, "/api/v1/post-its", card), http.StatusCreated, &postit)
	if postit.RunAs != ana.id {
		t.Errorf("run_as = %q, want Ana", postit.RunAs)
	}

	resource, _ := url.Parse(stub.URL)
	if err := s.db.UpdatePostIt(postit.Id, map[string]any{"resource": resource}); err != nil {
		t.Fatalf("retarget: %v", err)
	}

	// Any member may look at the card; it always runs with Ana's key.
	var result map[string]any
	s.mustJSON(owner.do(http.MethodGet, "/api/v1/post-its/"+postit.Id.String(), nil), http.StatusOK, &result)
	if len(seenKeys) != 1 || seenKeys[0] != anasKey {
		t.Errorf("provider saw %v, want Ana's key once", seenKeys)
	}

	// The owner edits the card: it now runs as them, and Ana's key is out of reach.
	if w := owner.do(http.MethodPatch, "/api/v1/post-its/"+postit.Id.String()+"/settings",
		map[string]any{"params": map[string]string{"$base": "EUR"}}); w.Code != http.StatusForbidden {
		t.Errorf("the owner rebound the card to Ana's secret: %d %s", w.Code, w.Body.String())
	}

	// Ana leaves: her scope is destroyed and her card stops.
	s.mustJSON(ana.do(http.MethodDelete, base+"/members/"+ana.id, nil), http.StatusNoContent, nil)
	if _, err := s.db.FindActiveKey(models.MemberScope(board.Id, ana.id)); err == nil {
		t.Error("Ana's data key survived her removal")
	}
	if err := s.cache.DropPostItResult(postit.Id); err != nil {
		t.Fatalf("drop cache: %v", err)
	}
	if w := owner.do(http.MethodGet, "/api/v1/post-its/"+postit.Id.String(), nil); w.Code != http.StatusForbidden {
		t.Errorf("the card still runs after Ana left: %d %s", w.Code, w.Body.String())
	}
	if w := ana.do(http.MethodGet, base, nil); w.Code != http.StatusNotFound {
		t.Errorf("Ana still sees the board she left: %d", w.Code)
	}

	s.assertNoResponseContains(anasKey)
}

// A group's secret follows its members: usable on any board they belong to,
// gone for them the moment they leave, gone for everyone when the group is.
func TestEndToEnd_GroupSecretFollowsItsMembers(t *testing.T) {
	original := safehttp.IsSafeIP
	safehttp.IsSafeIP = func(net.IP) bool { return true }
	t.Cleanup(func() { safehttp.IsSafeIP = original })

	const (
		opsKey  = "cur_live_ONLY_THE_OPS_GROUP_MAY_SEND_THIS"
		keyName = "OPS_CURRENCY_KEY"
	)

	var seenKeys []string
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenKeys = append(seenKeys, r.Header.Get("apikey"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"meta":{"last_updated_at":"2026-09-20T00:00:00Z"},"data":{"ARS":{"code":"ARS","value":1465.5}}}`)
	}))
	defer stub.Close()

	s := newStack(t)
	owner, ana := s.signUp("owner"), s.signUp("ana")

	var group models.Group
	s.mustJSON(owner.do(http.MethodPost, "/api/v1/groups", map[string]string{"name": "ops"}), http.StatusCreated, &group)
	groupBase := "/api/v1/groups/" + group.Id.String()
	defer owner.do(http.MethodDelete, groupBase, nil)

	s.mustJSON(owner.do(http.MethodPut, groupBase+"/secrets",
		map[string]string{"name": keyName, "kind": "api_key", "value": opsKey}), http.StatusNoContent, nil)
	s.mustJSON(owner.do(http.MethodPost, groupBase+"/members", map[string]string{"member": ana.id}), http.StatusNoContent, nil)

	// Ana's own board: the owner of the group is not on it, the key still is.
	board := ana.newBoard("anas")
	defer ana.do(http.MethodDelete, "/api/v1/boards/"+board.Id.String(), nil)

	var usable []models.SecretMeta
	s.mustJSON(ana.do(http.MethodGet, "/api/v1/boards/"+board.Id.String()+"/secrets/usable", nil), http.StatusOK, &usable)
	if len(usable) != 1 || usable[0].Name != keyName || usable[0].Scope != models.GroupScope(group.Id) {
		t.Errorf("Ana's usable secrets = %+v", usable)
	}

	var postit models.PostIts
	s.mustJSON(ana.do(http.MethodPost, "/api/v1/post-its", map[string]any{
		"board":      board.Id,
		"well_known": "exchange_rate",
		"params":     map[string]string{"$credential": "$" + keyName},
		"bindings":   map[string]any{keyName: map[string]any{"scope": models.GroupScope(group.Id), "name": keyName}},
	}), http.StatusCreated, &postit)

	resource, _ := url.Parse(stub.URL)
	if err := s.db.UpdatePostIt(postit.Id, map[string]any{"resource": resource}); err != nil {
		t.Fatalf("retarget: %v", err)
	}

	var result map[string]any
	s.mustJSON(ana.do(http.MethodGet, "/api/v1/post-its/"+postit.Id.String(), nil), http.StatusOK, &result)
	if len(seenKeys) != 1 || seenKeys[0] != opsKey {
		t.Errorf("provider saw %v, want the group's key once", seenKeys)
	}

	// Ana leaves the group: the card stops, the secret stays with the group.
	s.mustJSON(owner.do(http.MethodDelete, groupBase+"/members", map[string]string{"member": ana.id}), http.StatusNoContent, nil)
	if err := s.cache.DropPostItResult(postit.Id); err != nil {
		t.Fatalf("drop cache: %v", err)
	}
	if w := ana.do(http.MethodGet, "/api/v1/post-its/"+postit.Id.String(), nil); w.Code != http.StatusForbidden {
		t.Errorf("the card still runs after Ana left the group: %d %s", w.Code, w.Body.String())
	}
	if _, err := s.db.FindActiveKey(models.GroupScope(group.Id)); err != nil {
		t.Errorf("the group's data key is gone: %v", err)
	}

	// Deleting the group is crypto-shredding for everything it owned.
	s.mustJSON(owner.do(http.MethodDelete, groupBase, nil), http.StatusNoContent, nil)
	if _, err := s.db.FindActiveKey(models.GroupScope(group.Id)); err == nil {
		t.Error("the group's data key survived the delete")
	}

	s.assertNoResponseContains(opsKey)
}

// A profile secret shared with one user on one board: usable there and
// nowhere else, and gone the moment the share is withdrawn.
func TestEndToEnd_SharedSecretStopsWhenRevoked(t *testing.T) {
	original := safehttp.IsSafeIP
	safehttp.IsSafeIP = func(net.IP) bool { return true }
	t.Cleanup(func() { safehttp.IsSafeIP = original })

	const (
		alicesKey = "cur_live_ALICE_SHARED_THIS_WITH_BOB_ONLY"
		keyName   = "ALICES_CURRENCY_KEY"
	)

	var seenKeys []string
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenKeys = append(seenKeys, r.Header.Get("apikey"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"meta":{"last_updated_at":"2026-09-20T00:00:00Z"},"data":{"ARS":{"code":"ARS","value":1465.5}}}`)
	}))
	defer stub.Close()

	s := newStack(t)
	alice, bob := s.signUp("alice"), s.signUp("bob")

	shared, other := bob.newBoard("shared"), bob.newBoard("other")
	defer bob.do(http.MethodDelete, "/api/v1/boards/"+shared.Id.String(), nil)
	defer bob.do(http.MethodDelete, "/api/v1/boards/"+other.Id.String(), nil)

	profile := "/api/v1/users/" + alice.id + "/secrets"
	s.mustJSON(alice.do(http.MethodPut, profile,
		map[string]string{"name": keyName, "kind": "api_key", "value": alicesKey}), http.StatusNoContent, nil)
	defer alice.do(http.MethodDelete, profile+"/"+keyName, nil)

	if w := bob.do(http.MethodGet, profile, nil); w.Code != http.StatusNotFound {
		t.Errorf("Bob listed Alice's profile secrets: %d", w.Code)
	}

	grant := models.Grant{To: models.Audience{Kind: models.AudienceUser, ID: bob.id}, Board: shared.Id.String()}
	s.mustJSON(alice.do(http.MethodPut, profile+"/"+keyName+"/grants",
		map[string]any{"grants": []models.Grant{grant}}), http.StatusNoContent, nil)

	usableOn := func(board models.Board) []models.SecretMeta {
		var usable []models.SecretMeta
		s.mustJSON(bob.do(http.MethodGet, "/api/v1/boards/"+board.Id.String()+"/secrets/usable", nil), http.StatusOK, &usable)
		return usable
	}
	if usable := usableOn(shared); len(usable) != 1 || usable[0].Name != keyName || usable[0].Scope != models.UserScope(alice.id) {
		t.Errorf("Bob's usable secrets on the shared board = %+v", usable)
	}
	if usable := usableOn(other); len(usable) != 0 {
		t.Errorf("Bob's usable secrets on the other board = %+v, want nothing", usable)
	}

	card := func(board models.Board) map[string]any {
		return map[string]any{
			"board":      board.Id,
			"well_known": "exchange_rate",
			"params":     map[string]string{"$credential": "$" + keyName},
			"bindings":   map[string]any{keyName: map[string]any{"scope": models.UserScope(alice.id), "name": keyName}},
		}
	}
	if w := bob.do(http.MethodPost, "/api/v1/post-its", card(other)); w.Code != http.StatusForbidden {
		t.Errorf("Bob bound the key on the other board: %d %s", w.Code, w.Body.String())
	}

	var postit models.PostIts
	s.mustJSON(bob.do(http.MethodPost, "/api/v1/post-its", card(shared)), http.StatusCreated, &postit)

	resource, _ := url.Parse(stub.URL)
	if err := s.db.UpdatePostIt(postit.Id, map[string]any{"resource": resource}); err != nil {
		t.Fatalf("retarget: %v", err)
	}

	var result map[string]any
	s.mustJSON(bob.do(http.MethodGet, "/api/v1/post-its/"+postit.Id.String(), nil), http.StatusOK, &result)
	if len(seenKeys) != 1 || seenKeys[0] != alicesKey {
		t.Errorf("provider saw %v, want Alice's key once", seenKeys)
	}

	// Alice withdraws the share: Bob's card stops, Alice's own use is untouched.
	s.mustJSON(alice.do(http.MethodPut, profile+"/"+keyName+"/grants",
		map[string]any{"grants": []models.Grant{}}), http.StatusNoContent, nil)
	if err := s.cache.DropPostItResult(postit.Id); err != nil {
		t.Fatalf("drop cache: %v", err)
	}
	if w := bob.do(http.MethodGet, "/api/v1/post-its/"+postit.Id.String(), nil); w.Code != http.StatusForbidden {
		t.Errorf("Bob's card still runs after the share was withdrawn: %d %s", w.Code, w.Body.String())
	}
	if usable := usableOn(shared); len(usable) != 0 {
		t.Errorf("Bob still sees %+v after the share was withdrawn", usable)
	}

	s.assertNoResponseContains(alicesKey)
}

// A board shared by role: a viewer looks and runs, an editor builds with their
// own credentials, a demotion stops what they built with them, and removal
// takes the board and what they kept on it away.
func TestEndToEnd_BoardRoles(t *testing.T) {
	original := safehttp.IsSafeIP
	safehttp.IsSafeIP = func(net.IP) bool { return true }
	t.Cleanup(func() { safehttp.IsSafeIP = original })

	const (
		bsKey   = "cur_live_ONLY_BS_CARD_MAY_SEND_THIS"
		keyName = "BS_CURRENCY_KEY"
	)

	var seenKeys []string
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenKeys = append(seenKeys, r.Header.Get("apikey"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"meta":{"last_updated_at":"2026-09-20T00:00:00Z"},"data":{"ARS":{"code":"ARS","value":1465.5}}}`)
	}))
	defer stub.Close()
	resource, _ := url.Parse(stub.URL)

	s := newStack(t)
	owner, b := s.signUp("owner"), s.signUp("b")

	board := owner.newBoard("roles")
	defer owner.do(http.MethodDelete, "/api/v1/boards/"+board.Id.String(), nil)
	base := "/api/v1/boards/" + board.Id.String()

	var ownersCard models.PostIts
	s.mustJSON(owner.do(http.MethodPost, "/api/v1/post-its", map[string]any{"board": board.Id, "well_known": "dolar_oficial"}), http.StatusCreated, &ownersCard)
	if err := s.db.UpdatePostIt(ownersCard.Id, map[string]any{"resource": resource}); err != nil {
		t.Fatalf("retarget: %v", err)
	}
	ownersCardPath := "/api/v1/post-its/" + ownersCard.Id.String()

	// Shared as viewer: B lists and opens the board and reads results.
	owner.share(board, b, models.BoardViewer)

	var listed []models.Board
	s.mustJSON(b.do(http.MethodGet, "/api/v1/boards", nil), http.StatusOK, &listed)
	if len(listed) != 1 || listed[0].Id != board.Id || listed[0].Role != models.BoardViewer {
		t.Fatalf("B's boards = %+v, want the board as viewer", listed)
	}
	var members []models.BoardMemberSummary
	s.mustJSON(b.do(http.MethodGet, base+"/members", nil), http.StatusOK, &members)
	if len(members) != 2 || members[0].User.Id.String() != owner.id || members[1].User.Email != b.email || members[1].Role != models.BoardViewer {
		t.Errorf("members = %+v", members)
	}
	s.mustJSON(b.do(http.MethodGet, ownersCardPath, nil), http.StatusOK, nil)

	// ...and changes nothing, nor learns the board's credentials.
	for _, c := range []struct {
		method, path string
		body         any
	}{
		{http.MethodPost, "/api/v1/post-its", map[string]any{"board": board.Id, "well_known": "dolar_oficial"}},
		{http.MethodPatch, ownersCardPath + "/position", map[string]any{"x": 1, "y": 2}},
		{http.MethodPatch, ownersCardPath + "/settings", map[string]any{"rate": 5}},
		{http.MethodDelete, ownersCardPath, nil},
		{http.MethodGet, base + "/secrets", nil},
		{http.MethodGet, base + "/secrets/usable", nil},
		{http.MethodPut, base + "/members/" + b.id + "/secrets", map[string]string{"name": keyName, "kind": "api_key", "value": bsKey}},
		{http.MethodPut, base + "/members", map[string]any{"email": b.email, "role": "editor"}},
	} {
		if w := b.do(c.method, c.path, c.body); w.Code != http.StatusNotFound {
			t.Errorf("viewer %s %s = %d, want 404 (body: %s)", c.method, c.path, w.Code, w.Body.String())
		}
	}

	// Promoted to editor: B keeps a key on the board and builds a card with it.
	owner.share(board, b, models.BoardEditor)
	s.mustJSON(b.do(http.MethodPut, base+"/members/"+b.id+"/secrets",
		map[string]string{"name": keyName, "kind": "api_key", "value": bsKey}), http.StatusNoContent, nil)

	var bsCard models.PostIts
	s.mustJSON(b.do(http.MethodPost, "/api/v1/post-its", map[string]any{
		"board":      board.Id,
		"well_known": "exchange_rate",
		"params":     map[string]string{"$credential": "$" + keyName},
		"bindings":   map[string]any{keyName: map[string]any{"scope": models.MemberScope(board.Id, b.id), "name": keyName}},
	}), http.StatusCreated, &bsCard)
	if err := s.db.UpdatePostIt(bsCard.Id, map[string]any{"resource": resource}); err != nil {
		t.Fatalf("retarget: %v", err)
	}
	bsCardPath := "/api/v1/post-its/" + bsCard.Id.String()

	s.mustJSON(owner.do(http.MethodGet, bsCardPath, nil), http.StatusOK, nil)
	if len(seenKeys) == 0 || seenKeys[len(seenKeys)-1] != bsKey {
		t.Errorf("provider saw %v, want B's key last", seenKeys)
	}

	// Demoted to viewer: B may no longer bind, so the card that runs as B
	// stops, for everyone, until someone who may edits it.
	owner.share(board, b, models.BoardViewer)
	if err := s.cache.DropPostItResult(bsCard.Id); err != nil {
		t.Fatalf("drop cache: %v", err)
	}
	if w := owner.do(http.MethodGet, bsCardPath, nil); w.Code != http.StatusForbidden {
		t.Errorf("B's card still runs after the demotion: %d %s", w.Code, w.Body.String())
	}

	// Removed: the board leaves B's listing and B's key is destroyed.
	s.mustJSON(owner.do(http.MethodDelete, base+"/members/"+b.id, nil), http.StatusNoContent, nil)
	s.mustJSON(b.do(http.MethodGet, "/api/v1/boards", nil), http.StatusOK, &listed)
	if len(listed) != 0 {
		t.Errorf("B still lists %+v", listed)
	}
	if _, err := s.db.FindActiveKey(models.MemberScope(board.Id, b.id)); err == nil {
		t.Error("B's data key survived the removal")
	}

	// Nobody can be made a second owner, and an unknown address is reported.
	for body, want := range map[string]int{
		`{"email":"` + owner.email + `","role":"editor"}`: http.StatusBadRequest,
		`{"email":"` + b.email + `","role":"owner"}`:      http.StatusBadRequest,
		`{"email":"nobody@it.test","role":"viewer"}`:      http.StatusNotFound,
	} {
		var payload map[string]any
		_ = json.Unmarshal([]byte(body), &payload)
		if w := owner.do(http.MethodPut, base+"/members", payload); w.Code != want {
			t.Errorf("PUT members %s = %d, want %d (body: %s)", body, w.Code, want, w.Body.String())
		}
	}

	s.assertNoResponseContains(bsKey)
}

// An organization's boards and groups reach its members without being shared
// one by one: admins act as owners, members look, and the org cannot be
// deleted while it still owns something.
func TestEndToEnd_OrgBoardsAndGroups(t *testing.T) {
	s := newStack(t)
	admin, member, stranger := s.signUp("org-admin"), s.signUp("org-member"), s.signUp("stranger")

	var org models.Org
	s.mustJSON(admin.do(http.MethodPost, "/api/v1/orgs", map[string]string{"name": "acme"}), http.StatusCreated, &org)
	orgBase := "/api/v1/orgs/" + org.Id.String()
	s.mustJSON(admin.do(http.MethodPut, orgBase+"/members", map[string]string{"email": member.email, "role": "member"}), http.StatusOK, nil)

	var board models.Board
	s.mustJSON(admin.do(http.MethodPost, "/api/v1/boards", map[string]any{"name": "roadmap", "org": org.Id}), http.StatusCreated, &board)
	base := "/api/v1/boards/" + board.Id.String()
	if w := stranger.do(http.MethodPost, "/api/v1/boards", map[string]any{"name": "x", "org": org.Id}); w.Code != http.StatusNotFound {
		t.Errorf("a stranger created a board in the org: %d", w.Code)
	}

	// The member lists and opens the board as viewer, and cannot change it.
	var listed []models.Board
	s.mustJSON(member.do(http.MethodGet, "/api/v1/boards", nil), http.StatusOK, &listed)
	if len(listed) != 1 || listed[0].Id != board.Id || listed[0].Role != models.BoardViewer || listed[0].Org == nil {
		t.Fatalf("member's boards = %+v, want the org's board as viewer", listed)
	}
	s.mustJSON(stranger.do(http.MethodGet, "/api/v1/boards", nil), http.StatusOK, &listed)
	if len(listed) != 0 {
		t.Errorf("stranger's boards = %+v", listed)
	}
	newCard := map[string]any{"board": board.Id, "well_known": "dolar_oficial"}
	if w := member.do(http.MethodPost, "/api/v1/post-its", newCard); w.Code != http.StatusNotFound {
		t.Errorf("an org member created a card: %d", w.Code)
	}

	// Promoting them is an explicit role on the board.
	admin.share(board, member, models.BoardEditor)
	s.mustJSON(member.do(http.MethodPost, "/api/v1/post-its", newCard), http.StatusCreated, nil)

	// The org's group: its admin keeps a secret in it; the member sees the
	// group and nothing in it.
	var group models.Group
	s.mustJSON(admin.do(http.MethodPost, "/api/v1/groups", map[string]any{"name": "ops", "org": org.Id}), http.StatusCreated, &group)
	groupBase := "/api/v1/groups/" + group.Id.String()
	s.mustJSON(admin.do(http.MethodPut, groupBase+"/secrets",
		map[string]string{"name": "OPS_KEY", "kind": "api_key", "value": "ops-secret"}), http.StatusNoContent, nil)

	var seen models.Group
	s.mustJSON(member.do(http.MethodGet, groupBase, nil), http.StatusOK, &seen)
	if seen.Role != models.GroupViewer {
		t.Errorf("member's role in the group = %q, want viewer", seen.Role)
	}
	if w := member.do(http.MethodGet, groupBase+"/secrets", nil); w.Code != http.StatusNotFound {
		t.Errorf("an org member listed the group's secrets: %d", w.Code)
	}
	if w := stranger.do(http.MethodGet, groupBase, nil); w.Code != http.StatusNotFound {
		t.Errorf("a stranger saw the group: %d", w.Code)
	}

	usableBy := func(c *client) []models.SecretMeta {
		var usable []models.SecretMeta
		s.mustJSON(c.do(http.MethodGet, base+"/secrets/usable", nil), http.StatusOK, &usable)
		return usable
	}
	if usable := usableBy(admin); len(usable) != 1 || usable[0].Scope != models.GroupScope(group.Id) {
		t.Errorf("admin's usable = %+v, want the org group's key", usable)
	}
	if usable := usableBy(member); len(usable) != 0 {
		t.Errorf("member's usable = %+v, want nothing", usable)
	}

	// The org goes only once nothing belongs to it.
	for _, remove := range []string{base, groupBase} {
		if w := admin.do(http.MethodDelete, orgBase, nil); w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "org_not_empty") {
			t.Errorf("deleting the org with something in it: %d %s", w.Code, w.Body.String())
		}
		s.mustJSON(admin.do(http.MethodDelete, remove, nil), http.StatusNoContent, nil)
	}
	if w := member.do(http.MethodDelete, orgBase, nil); w.Code != http.StatusNotFound {
		t.Errorf("a member deleted the org: %d", w.Code)
	}
	s.mustJSON(admin.do(http.MethodDelete, orgBase, nil), http.StatusNoContent, nil)

	s.assertNoResponseContains("ops-secret")
}
