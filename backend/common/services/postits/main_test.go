package postits

import (
	"errors"
	"net/url"
	"slices"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/services/access"
	"github.com/google/uuid"
)

func newService(db *mocks.MockDB, cache *mocks.MockCache, run *mocks.MockExecuter) *PostItsService {
	if db == nil {
		db = &mocks.MockDB{}
	}
	if db.FindBoardFn == nil {
		db.FindBoardFn = boardOf(boardOwner).FindBoardFn
	}
	if cache == nil {
		cache = &mocks.MockCache{}
	}
	if run == nil {
		run = &mocks.MockExecuter{}
	}
	return New(db, cache, run, &mocks.MockSecretResolver{}, access.New())
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
	_, err := svc.CreatePostIt(boardOwner, &models.PostIts{Board: board})
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
	_, err := svc.CreatePostIt(boardOwner, &models.PostIts{Board: board, WellKnown: "dolar_oficial"})
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
	if len(got.Query) == 0 {
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
	_, err := svc.CreatePostIt(boardOwner, &models.PostIts{Board: uuid.New(), WellKnown: "events_search"})
	if err == nil {
		t.Fatal("expected error for missing required param, got nil")
	}
	if called {
		t.Error("db.CreatePostIt should not be called when param validation fails")
	}
}

func TestCreatePostIt_UnknownWellKnown(t *testing.T) {
	svc := newService(nil, nil, nil)
	_, err := svc.CreatePostIt(boardOwner, &models.PostIts{Board: uuid.New(), WellKnown: "does_not_exist"})
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
	if err := svc.UpdatePostIt(boardOwner, uuid.New(), map[string]any{}); err != nil {
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
		FindPostItFn: existingCard(id),
		UpdatePostItFn: func(id uuid.UUID, set map[string]any) error {
			gotID = id
			gotSet = set
			return nil
		},
	}

	svc := newService(db, nil, nil)
	set := map[string]any{"rate": 5}
	if err := svc.UpdatePostIt(boardOwner, id, set); err != nil {
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
	if err := svc.MovePostIt(boardOwner, id, models.Position{X: 1, Y: 2}); err != nil {
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
	if err := svc.MovePostIt(boardOwner, uuid.New(), models.Position{}); err == nil {
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

	postit, err := newService(db, nil, nil).GetPostIt(boardOwner, id)
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
		FindPostItFn: existingCard(id),
		DeletePostItFn: func(p uuid.UUID) (s []models.Strand, _ error) {
			if p != id {
				t.Errorf("id = %v, want %v", p, id)
			}
			called = true
			return s, nil
		},
	}

	if _, err := newService(db, nil, nil).DeletePostIt(boardOwner, id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("DeletePostIt was not delegated")
	}
}

// Reading, moving and deleting a card is for the members of its board; to
// anyone else the card does not exist.
func TestCardRoutes_AdmitMembersOnly(t *testing.T) {
	id := uuid.New()
	db := &mocks.MockDB{
		FindPostItFn:   existingCard(id),
		MovePostItFn:   func(_, _ uuid.UUID, _ models.Position) error { return nil },
		DeletePostItFn: func(uuid.UUID) ([]models.Strand, error) { return nil, nil },
	}
	svc := newService(db, nil, nil)

	calls := map[string]func(models.Principal) error{
		"get": func(p models.Principal) error {
			_, err := svc.GetPostIt(p, id)
			return err
		},
		"move": func(p models.Principal) error { return svc.MovePostIt(p, id, models.Position{}) },
		"delete": func(p models.Principal) error {
			_, err := svc.DeletePostIt(p, id)
			return err
		},
	}

	for name, call := range calls {
		if err := call(models.Principal{ID: "eve"}); !errors.Is(err, ErrNotAMember) {
			t.Errorf("%s as an outsider: got %v, want ErrNotAMember", name, err)
		}
		if err := call(boardOwner); err != nil {
			t.Errorf("%s as the owner: %v", name, err)
		}
	}
}

// A viewer looks at the cards and their results; changing them takes an editor.
func TestViewer_ReadsAndRunsButCannotChangeCards(t *testing.T) {
	id := uuid.New()
	viewer := models.Principal{ID: "viewer-id"}
	db := &mocks.MockDB{
		FindBoardFn: func(board uuid.UUID) (*models.Board, error) {
			return &models.Board{Id: board, Owner: boardOwner.ID, Members: []models.BoardMember{{User: viewer.ID, Role: models.BoardViewer}}}, nil
		},
		FindPostItFn: existingCard(id),
		CreatePostItFn: func(*models.PostIts, string, models.Position) (*models.PostIts, error) {
			t.Error("a viewer created a card")
			return nil, nil
		},
		UpdatePostItFn: func(uuid.UUID, map[string]any) error {
			t.Error("a viewer edited a card")
			return nil
		},
		MovePostItFn: func(_, _ uuid.UUID, _ models.Position) error {
			t.Error("a viewer moved a card")
			return nil
		},
		DeletePostItFn: func(uuid.UUID) ([]models.Strand, error) {
			t.Error("a viewer deleted a card")
			return nil, nil
		},
	}
	run := &mocks.MockExecuter{ExecuteFn: func(*models.PostIts) (any, error) { return "data", nil }}
	svc := New(db, &mocks.MockCache{}, run, &mocks.MockSecretResolver{}, access.New())

	postit, err := svc.GetPostIt(viewer, id)
	if err != nil {
		t.Fatalf("a viewer could not read the card: %v", err)
	}
	if data, err := svc.ExecutePostIt(postit); err != nil || data != "data" {
		t.Errorf("a viewer could not run the card: %v %v", data, err)
	}

	changes := map[string]error{}
	_, changes["create"] = svc.CreatePostIt(viewer, &models.PostIts{Board: uuid.New()})
	changes["update"] = svc.UpdatePostIt(viewer, id, map[string]any{"rate": 5})
	changes["move"] = svc.MovePostIt(viewer, id, models.Position{})
	_, changes["delete"] = svc.DeletePostIt(viewer, id)
	for name, err := range changes {
		if !errors.Is(err, ErrNotAMember) {
			t.Errorf("%s as a viewer: got %v, want ErrNotAMember", name, err)
		}
	}
}

func TestGetPostIt_MissingCardIsNotAMember(t *testing.T) {
	db := &mocks.MockDB{
		FindPostItFn: func(uuid.UUID) (*models.PostIts, error) { return nil, errors.New("mongo: no documents in result") },
	}
	if _, err := newService(db, nil, nil).GetPostIt(boardOwner, uuid.New()); !errors.Is(err, ErrNotAMember) {
		t.Errorf("got %v, want ErrNotAMember", err)
	}
}

func TestSecretRefs_OnlyMatchesUpperCaseNames(t *testing.T) {
	postit := &models.PostIts{
		Request: models.Request{
			Headers: map[string]string{"Authorization": "$API_KEY", "Accept": "application/json"},
			Queries: map[string]string{"lat": "$latitude", "token": "$TOKEN", "size": "1"},
		},
	}

	got := secretRefs(postit)
	slices.Sort(got)

	want := []string{"API_KEY", "TOKEN"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v ($latitude is a param, not a secret)", got, want)
	}
}

// The post-it the caller holds is serialised back to the client, so a resolved
// secret must never end up on it.
func TestExecutePostIt_DoesNotLeakSecretsOntoTheCaller(t *testing.T) {
	board := uuid.New()
	postit := &models.PostIts{
		Id:       uuid.New(),
		Board:    board,
		RunAs:    boardOwner.ID,
		Resource: &url.URL{Scheme: "https", Host: "example.com"},
		Request: models.Request{
			Method:  "GET",
			Headers: map[string]string{"Authorization": "$API_KEY"},
			Queries: map[string]string{"q": "$SEARCH"},
		},
	}

	resolver := &mocks.MockSecretResolver{
		ResolveAsFn: func(models.Principal, uuid.UUID, []models.SecretRef) (map[models.SecretRef]string, error) {
			return map[models.SecretRef]string{
				{Scope: models.BoardScope(board), Name: "API_KEY"}: "super-secret",
				{Scope: models.BoardScope(board), Name: "SEARCH"}:  "also-secret",
			}, nil
		},
	}

	var executed *models.PostIts
	run := &mocks.MockExecuter{
		ExecuteFn: func(p *models.PostIts) (any, error) {
			executed = p
			// Mimic the executer substituting in place.
			p.Request.Headers["Authorization"] = p.Params["$API_KEY"]
			return map[string]any{"ok": true}, nil
		},
	}

	svc := New(&mocks.MockDB{}, &mocks.MockCache{}, run, resolver, access.New())

	if _, err := svc.ExecutePostIt(postit); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if executed.Params["$API_KEY"] != "super-secret" {
		t.Error("the executer did not receive the resolved secret")
	}

	if _, present := postit.Params["$API_KEY"]; present {
		t.Error("the resolved secret was written onto the caller's post-it params")
	}
	if postit.Request.Headers["Authorization"] != "$API_KEY" {
		t.Errorf("the caller's headers were mutated to %q", postit.Request.Headers["Authorization"])
	}
}

// A post-it with no resource makes no outbound request, so it has no use for a
// secret. It also hands its params straight back to the caller, so resolving
// one would publish the plaintext. See the static_card well-known.
func TestExecutePostIt_ResourcelessPostItNeverResolvesSecrets(t *testing.T) {
	postit := &models.PostIts{
		Id:        uuid.New(),
		Board:     uuid.New(),
		WellKnown: "static_card",
		Params:    map[string]string{"text": "$API_KEY"},
		Resource:  nil,
	}

	resolved := false
	resolver := &mocks.MockSecretResolver{
		ResolveAsFn: func(models.Principal, uuid.UUID, []models.SecretRef) (map[models.SecretRef]string, error) {
			resolved = true
			return nil, nil
		},
	}

	run := &mocks.MockExecuter{
		ExecuteFn: func(p *models.PostIts) (any, error) {
			// What DewIt.Execute does when Resource is nil.
			return p.Params, nil
		},
	}

	svc := New(&mocks.MockDB{}, &mocks.MockCache{}, run, resolver, access.New())

	out, err := svc.ExecutePostIt(postit)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if resolved {
		t.Error("secrets were resolved for a post-it that makes no request")
	}

	params, ok := out.(map[string]string)
	if !ok {
		t.Fatalf("expected params map, got %T", out)
	}
	for key, value := range params {
		if value == "super-secret" {
			t.Errorf("decrypted secret returned to the caller under key %q", key)
		}
	}
	if params["text"] != "$API_KEY" {
		t.Errorf("text = %q, want the unresolved reference", params["text"])
	}
}

func TestUpdatePostIt_DropsTheCachedResult(t *testing.T) {
	id := uuid.New()

	var dropped uuid.UUID
	cache := &mocks.MockCache{
		DropPostItResultFn: func(got uuid.UUID) error {
			dropped = got
			return nil
		},
	}

	svc := newService(&mocks.MockDB{FindPostItFn: existingCard(id)}, cache, nil)

	if err := svc.UpdatePostIt(boardOwner, id, map[string]any{"rate": 60}); err != nil {
		t.Fatalf("update: %v", err)
	}

	if dropped != id {
		t.Errorf("dropped %v, want the edited post-it %v", dropped, id)
	}
}

func TestUpdatePostIt_KeepsCacheWhenTheWriteFails(t *testing.T) {
	id := uuid.New()
	db := &mocks.MockDB{
		FindPostItFn:   existingCard(id),
		UpdatePostItFn: func(uuid.UUID, map[string]any) error { return errors.New("boom") },
	}

	cache := &mocks.MockCache{
		DropPostItResultFn: func(uuid.UUID) error {
			t.Error("cache was dropped even though the update failed")
			return nil
		},
	}

	svc := newService(db, cache, nil)

	if err := svc.UpdatePostIt(boardOwner, id, map[string]any{"rate": 60}); err == nil {
		t.Fatal("expected the write error to surface")
	}
}

func existingCard(id uuid.UUID) func(uuid.UUID) (*models.PostIts, error) {
	return func(uuid.UUID) (*models.PostIts, error) {
		return &models.PostIts{Id: id, Board: uuid.New(), RunAs: boardOwner.ID}, nil
	}
}
