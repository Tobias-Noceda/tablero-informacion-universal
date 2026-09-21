package postits

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	srv "github.com/Secreto31126/tesis/common/services/postits"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const owner = "owner-id"

func setupRouter(db *mocks.MockDB, cache *mocks.MockCache, run *mocks.MockExecuter) *gin.Engine {
	return setupRouterWith(db, cache, run, &mocks.MockSecretResolver{})
}

func setupRouterWith(db *mocks.MockDB, cache *mocks.MockCache, run *mocks.MockExecuter, secrets *mocks.MockSecretResolver) *gin.Engine {
	if db == nil {
		db = &mocks.MockDB{}
	}
	if db.FindBoardFn == nil {
		db.FindBoardFn = func(id uuid.UUID) (*models.Board, error) {
			return &models.Board{Id: id, Owner: owner}, nil
		}
	}
	if db.FindPostItFn == nil {
		db.FindPostItFn = func(id uuid.UUID) (*models.PostIts, error) {
			return &models.PostIts{Id: id, Board: uuid.New(), RunAs: owner}, nil
		}
	}
	if cache == nil {
		cache = &mocks.MockCache{}
	}
	if run == nil {
		run = &mocks.MockExecuter{}
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewController(srv.New(db, cache, run, secrets)).RegisterRoutes(r)
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

func TestCreatePostIt_OK(t *testing.T) {
	board := uuid.New()
	newID := uuid.New()
	db := &mocks.MockDB{
		CreatePostItFn: func(p *models.PostIts, ptype string, pos models.Position) (*models.PostIts, error) {
			p.Id = newID
			return p, nil
		},
	}
	r := setupRouter(db, nil, nil)

	w := do(r, http.MethodPost, "/post-its", `{"cognito_id":"`+owner+`","board":"`+board.String()+`","well-known":"dolar_oficial"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", w.Code, w.Body.String())
	}

	var got struct {
		Id        uuid.UUID
		Board     uuid.UUID
		WellKnown string
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Id != newID {
		t.Errorf("id = %v, want %v", got.Id, newID)
	}
	if got.Board != board {
		t.Errorf("board = %v, want %v", got.Board, board)
	}
}

func TestCreatePostIt_MissingBoard(t *testing.T) {
	r := setupRouter(nil, nil, nil)
	w := do(r, http.MethodPost, "/post-its", `{"cognito_id":"`+owner+`","well-known":"dolar_oficial"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (board is required)", w.Code)
	}
}

func TestCreatePostIt_BadJSON(t *testing.T) {
	r := setupRouter(nil, nil, nil)
	w := do(r, http.MethodPost, "/post-its", `{not json`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestGetPostItSettings_OK(t *testing.T) {
	id := uuid.New()
	db := &mocks.MockDB{
		FindPostItFn: func(_ uuid.UUID) (*models.PostIts, error) {
			return &models.PostIts{Id: id, Rate: 5, Query: map[string]string{"compra": ".compra"}}, nil
		},
	}
	r := setupRouter(db, nil, nil)

	w := do(r, http.MethodGet, "/post-its/"+id.String()+"/settings", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var got struct {
		Rate  int
		Query map[string]string
	}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.Rate != 5 || got.Query["compra"] != ".compra" {
		t.Errorf("got %+v, want rate=5 and query[compra]=.compra", got)
	}
}

func TestGetPostItSettings_InvalidUUID(t *testing.T) {
	r := setupRouter(nil, nil, nil)
	w := do(r, http.MethodGet, "/post-its/not-a-uuid/settings", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestEditPostIt_BuildsSetMap(t *testing.T) {
	id := uuid.New()
	var gotSet map[string]any
	db := &mocks.MockDB{
		UpdatePostItFn: func(_ uuid.UUID, set map[string]any) error {
			gotSet = set
			return nil
		},
	}
	r := setupRouter(db, nil, nil)

	w := do(r, http.MethodPatch, "/post-its/"+id.String()+"/settings", `{"cognito_id":"`+owner+`","rate":5,"query":{"x":".x"}}`)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (body: %s)", w.Code, w.Body.String())
	}
	query, ok := gotSet["query"].(map[string]string)
	if gotSet["rate"] == nil || !ok || query["x"] != ".x" {
		t.Errorf("set = %v, want rate and query[x]=.x", gotSet)
	}
	if _, ok := gotSet["response"]; ok {
		t.Errorf("set should not contain response when omitted: %v", gotSet)
	}
}

func TestEditPostIt_EmptyBodyRejected(t *testing.T) {
	id := uuid.New()
	called := false
	db := &mocks.MockDB{
		UpdatePostItFn: func(_ uuid.UUID, _ map[string]any) error {
			called = true
			return nil
		},
	}
	r := setupRouter(db, nil, nil)

	w := do(r, http.MethodPatch, "/post-its/"+id.String()+"/settings", `{"cognito_id":"`+owner+`"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (no fields to update)", w.Code)
	}
	if called {
		t.Error("UpdatePostIt should not be called when no fields are provided")
	}
}

func TestMovePostIt_OK(t *testing.T) {
	id := uuid.New()
	var gotPos models.Position
	db := &mocks.MockDB{
		FindPostItFn: func(_ uuid.UUID) (*models.PostIts, error) {
			return &models.PostIts{Id: id, Board: uuid.New()}, nil
		},
		MovePostItFn: func(_, _ uuid.UUID, pos models.Position) error {
			gotPos = pos
			return nil
		},
	}
	r := setupRouter(db, nil, nil)

	w := do(r, http.MethodPatch, "/post-its/"+id.String()+"/position", `{"x":150,"y":250}`)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (body: %s)", w.Code, w.Body.String())
	}
	if gotPos.X != 150 || gotPos.Y != 250 {
		t.Errorf("pos = %+v, want {150 250}", gotPos)
	}
}

func TestExecutePostIt_OK(t *testing.T) {
	id := uuid.New()
	db := &mocks.MockDB{
		FindPostItFn: func(_ uuid.UUID) (*models.PostIts, error) {
			return &models.PostIts{Id: id}, nil
		},
	}
	run := &mocks.MockExecuter{
		ExecuteFn: func(_ *models.PostIts) (any, error) {
			return map[string]any{"compra": 1000.0}, nil
		},
	}
	r := setupRouter(db, nil, run)

	w := do(r, http.MethodGet, "/post-its/"+id.String(), "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var got map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got["compra"] != 1000.0 {
		t.Errorf("got %v, want compra=1000", got)
	}
}

func TestExecutePostIt_NotFound(t *testing.T) {
	db := &mocks.MockDB{
		FindPostItFn: func(_ uuid.UUID) (*models.PostIts, error) {
			return nil, errors.New("mongo: no documents in result")
		},
	}
	r := setupRouter(db, nil, nil)

	w := do(r, http.MethodGet, "/post-its/"+uuid.New().String(), "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestExecutePostIt_ExecError(t *testing.T) {
	db := &mocks.MockDB{
		FindPostItFn: func(_ uuid.UUID) (*models.PostIts, error) {
			return &models.PostIts{Id: uuid.New()}, nil
		},
	}
	run := &mocks.MockExecuter{
		ExecuteFn: func(_ *models.PostIts) (any, error) {
			return nil, errors.New("resource didn't return 200")
		},
	}
	r := setupRouter(db, nil, run)

	w := do(r, http.MethodGet, "/post-its/"+uuid.New().String(), "")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

func TestDeletePostIt_OK(t *testing.T) {
	id := uuid.New()
	called := false
	db := &mocks.MockDB{
		DeletePostItFn: func(_ uuid.UUID) ([]models.Strand, error) {
			called = true
			return []models.Strand{{Id: id, Source: id, Target: id}}, nil
		},
	}
	r := setupRouter(db, nil, nil)

	w := do(r, http.MethodDelete, "/post-its/"+uuid.New().String(), "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !called {
		t.Error("DeletePostIt was not called")
	}
	var got []struct{ Id uuid.UUID }
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got[0].Id != id {
		t.Errorf("id = %v, want %v", got[0].Id, id)
	}
}

func TestDeletePostIt_InvalidUUID(t *testing.T) {
	r := setupRouter(nil, nil, nil)
	w := do(r, http.MethodDelete, "/post-its/nope", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestDeletePostIt_ServiceError(t *testing.T) {
	db := &mocks.MockDB{
		DeletePostItFn: func(_ uuid.UUID) ([]models.Strand, error) {
			return nil, errors.New("mongo: no documents in result")
		},
	}
	r := setupRouter(db, nil, nil)

	w := do(r, http.MethodDelete, "/post-its/"+uuid.New().String(), "")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

func TestCreatePostIt_ServiceError(t *testing.T) {
	db := &mocks.MockDB{
		CreatePostItFn: func(_ *models.PostIts, _ string, _ models.Position) (*models.PostIts, error) {
			return nil, errors.New("mongo: no documents in result")
		},
	}
	r := setupRouter(db, nil, nil)

	w := do(r, http.MethodPost, "/post-its", `{"cognito_id":"`+owner+`","board":"`+uuid.New().String()+`"}`)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

func TestEditPostIt_InvalidUUID(t *testing.T) {
	r := setupRouter(nil, nil, nil)
	w := do(r, http.MethodPatch, "/post-its/nope/settings", `{"cognito_id":"`+owner+`","rate":1}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestEditPostIt_ServiceError(t *testing.T) {
	db := &mocks.MockDB{
		UpdatePostItFn: func(_ uuid.UUID, _ map[string]any) error {
			return errors.New("mongo: no documents in result")
		},
	}
	r := setupRouter(db, nil, nil)

	w := do(r, http.MethodPatch, "/post-its/"+uuid.New().String()+"/settings", `{"cognito_id":"`+owner+`","rate":1}`)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

func TestMovePostIt_InvalidUUID(t *testing.T) {
	r := setupRouter(nil, nil, nil)
	w := do(r, http.MethodPatch, "/post-its/nope/position", `{"x":1,"y":2}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestMovePostIt_MissingCoords(t *testing.T) {
	r := setupRouter(nil, nil, nil)
	// x/y are required,number -- a zero/absent value must be rejected.
	w := do(r, http.MethodPatch, "/post-its/"+uuid.New().String()+"/position", `{}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func mustURL(raw string) *url.URL {
	parsed, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return parsed
}

// A card whose platform credential is not provisioned is a service problem,
// not a client error, and the response must not say which credential.
func TestExecutePostIt_MissingSystemSecretIsUnavailable(t *testing.T) {
	db := &mocks.MockDB{
		FindPostItFn: func(_ uuid.UUID) (*models.PostIts, error) {
			return &models.PostIts{Id: uuid.New(), WellKnown: "nasa_apod", Resource: mustURL("https://api.nasa.gov/planetary/apod")}, nil
		},
	}
	r := setupRouter(db, nil, nil)

	w := do(r, http.MethodGet, "/post-its/"+uuid.New().String(), "")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (body: %s)", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "NASA") {
		t.Errorf("the response names the missing secret: %s", w.Body.String())
	}
}

func TestCreatePostIt_RequiresACaller(t *testing.T) {
	r := setupRouter(nil, nil, nil)
	w := do(r, http.MethodPost, "/post-its", `{"board":"`+uuid.New().String()+`"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (cognito_id is required)", w.Code)
	}
}

func TestCreatePostIt_OutsiderIsForbidden(t *testing.T) {
	r := setupRouter(nil, nil, nil)
	w := do(r, http.MethodPost, "/post-its", `{"cognito_id":"someone-else","board":"`+uuid.New().String()+`"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (body: %s)", w.Code, w.Body.String())
	}
}

func TestCreatePostIt_BindsAsTheCaller(t *testing.T) {
	board := uuid.New()
	var gotPrincipal models.Principal
	var gotRefs []models.SecretRef
	secrets := &mocks.MockSecretResolver{
		CanBindFn: func(p models.Principal, _ uuid.UUID, refs []models.SecretRef) error {
			gotPrincipal, gotRefs = p, refs
			return nil
		},
	}
	db := &mocks.MockDB{
		CreatePostItFn: func(p *models.PostIts, _ string, _ models.Position) (*models.PostIts, error) {
			return p, nil
		},
	}
	r := setupRouterWith(db, nil, nil, secrets)

	body := `{"cognito_id":"` + owner + `","board":"` + board.String() + `","well_known":"exchange_rate",
	         "params":{"$credential":"$MINE"},
	         "bindings":{"MINE":{"scope":{"kind":"member","owner":"` + board.String() + `:` + owner + `"},"name":"MINE"}}}`
	w := do(r, http.MethodPost, "/post-its", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", w.Code, w.Body.String())
	}

	want := models.SecretRef{Scope: models.MemberScope(board, owner), Name: "MINE"}
	if gotPrincipal.ID != owner || len(gotRefs) != 1 || gotRefs[0] != want {
		t.Errorf("CanBind(%v, %v), want (%s, [%v])", gotPrincipal, gotRefs, owner, want)
	}

	var got struct {
		RunAs    string                      `json:"run_as"`
		Bindings map[string]models.SecretRef `json:"bindings"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.RunAs != owner || got.Bindings["MINE"] != want {
		t.Errorf("response run_as=%q bindings=%v", got.RunAs, got.Bindings)
	}
}

func TestCreatePostIt_ForbiddenBindingIs403(t *testing.T) {
	secrets := &mocks.MockSecretResolver{
		CanBindFn: func(models.Principal, uuid.UUID, []models.SecretRef) error {
			return infrastructure.ErrForbidden
		},
	}
	r := setupRouterWith(nil, nil, nil, secrets)

	board := uuid.New()
	body := `{"cognito_id":"` + owner + `","board":"` + board.String() + `",
	         "bindings":{"X":{"scope":{"kind":"board","owner":"` + board.String() + `"},"name":"X"}}}`
	w := do(r, http.MethodPost, "/post-its", body)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (body: %s)", w.Code, w.Body.String())
	}
}

func TestEditPostIt_RequiresACaller(t *testing.T) {
	r := setupRouter(nil, nil, nil)
	w := do(r, http.MethodPatch, "/post-its/"+uuid.New().String()+"/settings", `{"rate":1}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (cognito_id is required)", w.Code)
	}
}

func TestEditPostIt_ForwardsBindings(t *testing.T) {
	board := uuid.New()
	var gotSet map[string]any
	db := &mocks.MockDB{
		UpdatePostItFn: func(_ uuid.UUID, set map[string]any) error {
			gotSet = set
			return nil
		},
	}
	r := setupRouter(db, nil, nil)

	body := `{"cognito_id":"` + owner + `","bindings":{"MINE":{"scope":{"kind":"board","owner":"` + board.String() + `"},"name":"MINE"}}}`
	w := do(r, http.MethodPatch, "/post-its/"+uuid.New().String()+"/settings", body)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (body: %s)", w.Code, w.Body.String())
	}
	bindings, ok := gotSet["bindings"].(map[string]models.SecretRef)
	if !ok || bindings["MINE"].Name != "MINE" || gotSet["runas"] != owner {
		t.Errorf("set = %v, want bindings and runas", gotSet)
	}
}

func TestEditPostIt_OutsiderIsForbidden(t *testing.T) {
	r := setupRouter(nil, nil, nil)
	w := do(r, http.MethodPatch, "/post-its/"+uuid.New().String()+"/settings", `{"cognito_id":"someone-else","rate":1}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (body: %s)", w.Code, w.Body.String())
	}
}

func TestExecutePostIt_UnavailableCredentialIs403(t *testing.T) {
	id := uuid.New()
	db := &mocks.MockDB{
		FindPostItFn: func(uuid.UUID) (*models.PostIts, error) {
			return &models.PostIts{
				Id:       id,
				Board:    uuid.New(),
				RunAs:    owner,
				Resource: &url.URL{Scheme: "https", Host: "example.com"},
				Request:  models.Request{Headers: map[string]string{"apikey": "$KEY"}},
			}, nil
		},
	}
	secrets := &mocks.MockSecretResolver{
		ResolveAsFn: func(models.Principal, uuid.UUID, []models.SecretRef) (map[models.SecretRef]string, error) {
			return nil, infrastructure.ErrForbidden
		},
	}
	r := setupRouterWith(db, nil, nil, secrets)

	w := do(r, http.MethodGet, "/post-its/"+id.String(), "")
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (body: %s)", w.Code, w.Body.String())
	}
}
