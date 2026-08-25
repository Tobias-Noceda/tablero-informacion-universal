package postits

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	srv "github.com/Secreto31126/tesis/common/services/postits"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func setupRouter(db *mocks.MockDB, cache *mocks.MockCache, run *mocks.MockExecuter) *gin.Engine {
	if db == nil {
		db = &mocks.MockDB{}
	}
	if cache == nil {
		cache = &mocks.MockCache{}
	}
	if run == nil {
		run = &mocks.MockExecuter{}
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewController(srv.New(db, cache, run, &mocks.MockSecretResolver{})).RegisterRoutes(r)
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

	w := do(r, http.MethodPost, "/post-its", `{"board":"`+board.String()+`","well-known":"dolar_oficial"}`)
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
	w := do(r, http.MethodPost, "/post-its", `{"well-known":"dolar_oficial"}`)
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

	w := do(r, http.MethodPatch, "/post-its/"+id.String()+"/settings", `{"rate":5,"query":{"x":".x"}}`)
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

	w := do(r, http.MethodPatch, "/post-its/"+id.String()+"/settings", `{}`)
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
	called := false
	db := &mocks.MockDB{
		DeletePostItFn: func(_ uuid.UUID) error {
			called = true
			return nil
		},
	}
	r := setupRouter(db, nil, nil)

	w := do(r, http.MethodDelete, "/post-its/"+uuid.New().String(), "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
	if !called {
		t.Error("DeletePostIt was not called")
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
		DeletePostItFn: func(_ uuid.UUID) error {
			return errors.New("mongo: no documents in result")
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

	w := do(r, http.MethodPost, "/post-its", `{"board":"`+uuid.New().String()+`"}`)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

func TestEditPostIt_InvalidUUID(t *testing.T) {
	r := setupRouter(nil, nil, nil)
	w := do(r, http.MethodPatch, "/post-its/nope/settings", `{"rate":1}`)
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

	w := do(r, http.MethodPatch, "/post-its/"+uuid.New().String()+"/settings", `{"rate":1}`)
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
