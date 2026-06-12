package postits

import (
	"errors"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

func newService(db *mocks.MockDB, cache *mocks.MockCache, run *mocks.MockExecuter) *PostItsService {
	if db == nil {
		db = &mocks.MockDB{}
	}
	if cache == nil {
		cache = &mocks.MockCache{}
	}
	if run == nil {
		run = &mocks.MockExecuter{}
	}
	return New(db, cache, run)
}

func TestCreatePostIt_PlainPassesThrough(t *testing.T) {
	board := uuid.New()
	var gotType string
	var gotBoard uuid.UUID

	db := &mocks.MockDB{
		CreatePostItFn: func(p *models.PostIts, ptype string, pos models.Position) (*models.PostIts, error) {
			gotType = ptype
			gotBoard = p.Board
			return p, nil
		},
	}

	svc := newService(db, nil, nil)
	_, err := svc.CreatePostIt(&models.PostIts{Board: board})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotType != "" {
		t.Errorf("ptype = %q, want empty for a non well-known post-it", gotType)
	}
	if gotBoard != board {
		t.Errorf("board = %v, want %v", gotBoard, board)
	}
}

func TestCreatePostIt_ResolvesWellKnown(t *testing.T) {
	board := uuid.New()
	var got *models.PostIts
	var gotType string

	db := &mocks.MockDB{
		CreatePostItFn: func(p *models.PostIts, ptype string, pos models.Position) (*models.PostIts, error) {
			got = p
			gotType = ptype
			return p, nil
		},
	}

	svc := newService(db, nil, nil)
	_, err := svc.CreatePostIt(&models.PostIts{Board: board, WellKnown: "dolar_oficial"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotType != "dolar_oficial" {
		t.Errorf("ptype = %q, want dolar_oficial", gotType)
	}
	if got.Board != board {
		t.Errorf("board = %v, want %v (board must survive well-known resolution)", got.Board, board)
	}
	if got.Resource == nil || got.Resource.Host != "dolarapi.com" {
		t.Errorf("resource not populated from well-known: %+v", got.Resource)
	}
	if got.Query == "" {
		t.Error("query not populated from well-known")
	}
}

func TestCreatePostIt_MissingRequiredParam(t *testing.T) {
	called := false
	db := &mocks.MockDB{
		CreatePostItFn: func(p *models.PostIts, ptype string, pos models.Position) (*models.PostIts, error) {
			called = true
			return p, nil
		},
	}

	svc := newService(db, nil, nil)
	// events_search requires a $keyword param; none provided.
	_, err := svc.CreatePostIt(&models.PostIts{Board: uuid.New(), WellKnown: "events_search"})
	if err == nil {
		t.Fatal("expected error for missing required param, got nil")
	}
	if called {
		t.Error("db.CreatePostIt should not be called when param validation fails")
	}
}

func TestCreatePostIt_UnknownWellKnown(t *testing.T) {
	svc := newService(nil, nil, nil)
	_, err := svc.CreatePostIt(&models.PostIts{Board: uuid.New(), WellKnown: "does_not_exist"})
	if err == nil {
		t.Fatal("expected error for unknown well-known, got nil")
	}
}

func TestUpdatePostIt_EmptySetSkipsDB(t *testing.T) {
	called := false
	db := &mocks.MockDB{
		UpdatePostItFn: func(id uuid.UUID, set map[string]any) error {
			called = true
			return nil
		},
	}

	svc := newService(db, nil, nil)
	if err := svc.UpdatePostIt(uuid.New(), map[string]any{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("db.UpdatePostIt should not be called for an empty set")
	}
}

func TestUpdatePostIt_ForwardsSet(t *testing.T) {
	id := uuid.New()
	var gotID uuid.UUID
	var gotSet map[string]any

	db := &mocks.MockDB{
		UpdatePostItFn: func(id uuid.UUID, set map[string]any) error {
			gotID = id
			gotSet = set
			return nil
		},
	}

	svc := newService(db, nil, nil)
	set := map[string]any{"rate": 5}
	if err := svc.UpdatePostIt(id, set); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotID != id {
		t.Errorf("id = %v, want %v", gotID, id)
	}
	if gotSet["rate"] != 5 {
		t.Errorf("set = %v, want rate=5", gotSet)
	}
}

func TestMovePostIt_UsesBoardFromPostIt(t *testing.T) {
	id := uuid.New()
	board := uuid.New()
	var gotBoard, gotID uuid.UUID
	var gotPos models.Position

	db := &mocks.MockDB{
		FindPostItFn: func(_ uuid.UUID) (*models.PostIts, error) {
			return &models.PostIts{Id: id, Board: board}, nil
		},
		MovePostItFn: func(b, p uuid.UUID, pos models.Position) error {
			gotBoard, gotID, gotPos = b, p, pos
			return nil
		},
	}

	svc := newService(db, nil, nil)
	if err := svc.MovePostIt(id, models.Position{X: 1, Y: 2}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotBoard != board || gotID != id {
		t.Errorf("moved (board=%v,id=%v), want (board=%v,id=%v)", gotBoard, gotID, board, id)
	}
	if gotPos.X != 1 || gotPos.Y != 2 {
		t.Errorf("pos = %+v, want {1 2}", gotPos)
	}
}

func TestMovePostIt_FindError(t *testing.T) {
	db := &mocks.MockDB{
		FindPostItFn: func(_ uuid.UUID) (*models.PostIts, error) {
			return nil, errors.New("not found")
		},
	}
	svc := newService(db, nil, nil)
	if err := svc.MovePostIt(uuid.New(), models.Position{}); err == nil {
		t.Fatal("expected error propagated from FindPostIt")
	}
}

func TestExecutePostIt_CacheHit(t *testing.T) {
	executed := false
	cache := &mocks.MockCache{
		FindPostItResultFn: func(_ uuid.UUID) (any, error) {
			return "cached", nil
		},
	}
	run := &mocks.MockExecuter{
		ExecuteFn: func(_ *models.PostIts) (any, error) {
			executed = true
			return "fresh", nil
		},
	}

	svc := newService(nil, cache, run)
	data, err := svc.ExecutePostIt(&models.PostIts{Id: uuid.New()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data != "cached" {
		t.Errorf("data = %v, want cached", data)
	}
	if executed {
		t.Error("executer should not run on a cache hit")
	}
}

func TestExecutePostIt_CacheMissRunsExecuter(t *testing.T) {
	run := &mocks.MockExecuter{
		ExecuteFn: func(_ *models.PostIts) (any, error) {
			return "fresh", nil
		},
	}
	// Default MockCache returns ErrCacheMiss.
	svc := newService(nil, &mocks.MockCache{}, run)
	data, err := svc.ExecutePostIt(&models.PostIts{Id: uuid.New()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data != "fresh" {
		t.Errorf("data = %v, want fresh", data)
	}
}

func TestExecutePostIt_ExecuterError(t *testing.T) {
	run := &mocks.MockExecuter{
		ExecuteFn: func(_ *models.PostIts) (any, error) {
			return nil, errors.New("boom")
		},
	}
	svc := newService(nil, &mocks.MockCache{}, run)
	if _, err := svc.ExecutePostIt(&models.PostIts{Id: uuid.New()}); err == nil {
		t.Fatal("expected error propagated from executer")
	}
}

func TestGetPostIt_Delegates(t *testing.T) {
	id := uuid.New()
	var got uuid.UUID
	db := &mocks.MockDB{
		FindPostItFn: func(p uuid.UUID) (*models.PostIts, error) {
			got = p
			return &models.PostIts{Id: p}, nil
		},
	}

	postit, err := newService(db, nil, nil).GetPostIt(id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != id || postit.Id != id {
		t.Errorf("GetPostIt(%v) used %v / returned %v", id, got, postit.Id)
	}
}

func TestDeletePostIt_Delegates(t *testing.T) {
	id := uuid.New()
	called := false
	db := &mocks.MockDB{
		DeletePostItFn: func(p uuid.UUID) error {
			if p != id {
				t.Errorf("id = %v, want %v", p, id)
			}
			called = true
			return nil
		},
	}

	if err := newService(db, nil, nil).DeletePostIt(id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("DeletePostIt was not delegated")
	}
}
