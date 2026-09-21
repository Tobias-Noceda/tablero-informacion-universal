package postits

import (
	"errors"
	"net/url"
	"testing"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

var (
	boardOwner = models.Principal{ID: "owner-id"}
	member     = models.Principal{ID: "member-id"}
	outsider   = models.Principal{ID: "outsider-id"}
)

func boardOf(owner models.Principal, collaborators ...models.Principal) *mocks.MockDB {
	db := &mocks.MockDB{
		FindBoardFn: func(id uuid.UUID) (*models.Board, error) {
			board := &models.Board{Id: id, Owner: owner.ID}
			for _, c := range collaborators {
				board.Collaborators = append(board.Collaborators, c.ID)
			}
			return board, nil
		},
		CreatePostItFn: func(p *models.PostIts, _ string, _ models.Position) (*models.PostIts, error) {
			return p, nil
		},
	}
	return db
}

func TestCreatePostIt_RequiresMembership(t *testing.T) {
	db := boardOf(boardOwner)
	created := false
	db.CreatePostItFn = func(p *models.PostIts, _ string, _ models.Position) (*models.PostIts, error) {
		created = true
		return p, nil
	}
	svc := newService(db, nil, nil)

	_, err := svc.CreatePostIt(outsider, &models.PostIts{Board: uuid.New()})
	if !errors.Is(err, ErrNotAMember) {
		t.Errorf("got %v, want ErrNotAMember", err)
	}
	if created {
		t.Error("the post-it was stored anyway")
	}
}

func TestCreatePostIt_RunsAsItsCreator(t *testing.T) {
	svc := newService(boardOf(boardOwner, member), nil, nil)

	created, err := svc.CreatePostIt(member, &models.PostIts{Board: uuid.New()})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.RunAs != member.ID {
		t.Errorf("run_as = %q, want the creator", created.RunAs)
	}
}

func TestCreatePostIt_ValidatesBindingsAsTheCreator(t *testing.T) {
	board := uuid.New()
	mine := models.SecretRef{Scope: models.MemberScope(board, member.ID), Name: "MINE"}

	var gotPrincipal models.Principal
	var gotRefs []models.SecretRef
	resolver := &mocks.MockSecretResolver{
		CanBindFn: func(p models.Principal, b uuid.UUID, refs []models.SecretRef) error {
			gotPrincipal, gotRefs = p, refs
			return nil
		},
	}
	svc := New(boardOf(boardOwner, member), &mocks.MockCache{}, &mocks.MockExecuter{}, resolver)

	created, err := svc.CreatePostIt(member, &models.PostIts{
		Board:     board,
		WellKnown: "exchange_rate",
		Params:    map[string]string{"$credential": "$MINE"},
		Bindings:  map[string]models.SecretRef{"MINE": mine},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if gotPrincipal != member || len(gotRefs) != 1 || gotRefs[0] != mine {
		t.Errorf("CanBind(%v, %v), want (%v, [%v])", gotPrincipal, gotRefs, member, mine)
	}
	if created.Bindings["MINE"] != mine {
		t.Errorf("bindings = %v, want them kept on the well-known", created.Bindings)
	}
}

func TestCreatePostIt_RefusesWhatCannotBeBound(t *testing.T) {
	board := uuid.New()
	resolver := &mocks.MockSecretResolver{
		CanBindFn: func(models.Principal, uuid.UUID, []models.SecretRef) error {
			return infrastructure.ErrForbidden
		},
	}
	svc := New(boardOf(boardOwner, member), &mocks.MockCache{}, &mocks.MockExecuter{}, resolver)

	_, err := svc.CreatePostIt(member, &models.PostIts{
		Board:    board,
		Bindings: map[string]models.SecretRef{"THEIRS": {Scope: models.MemberScope(board, boardOwner.ID), Name: "THEIRS"}},
	})
	if !errors.Is(err, infrastructure.ErrForbidden) {
		t.Errorf("got %v, want ErrForbidden", err)
	}
}

func TestCreatePostIt_BindingAliasesAreSecretNames(t *testing.T) {
	board := uuid.New()
	svc := newService(boardOf(boardOwner), nil, nil)

	_, err := svc.CreatePostIt(boardOwner, &models.PostIts{
		Board:    board,
		Bindings: map[string]models.SecretRef{"lowercase": {Scope: models.BoardScope(board), Name: "KEY"}},
	})
	if !errors.Is(err, ErrInvalidBinding) {
		t.Errorf("got %v, want ErrInvalidBinding", err)
	}
}

func TestUpdatePostIt_RebindsAsTheEditor(t *testing.T) {
	board := uuid.New()
	id := uuid.New()
	mine := models.SecretRef{Scope: models.MemberScope(board, member.ID), Name: "MINE"}

	db := boardOf(boardOwner, member)
	db.FindPostItFn = func(uuid.UUID) (*models.PostIts, error) {
		return &models.PostIts{Id: id, Board: board, RunAs: boardOwner.ID}, nil
	}
	var gotSet map[string]any
	db.UpdatePostItFn = func(_ uuid.UUID, set map[string]any) error {
		gotSet = set
		return nil
	}
	var gotPrincipal models.Principal
	var gotRefs []models.SecretRef
	resolver := &mocks.MockSecretResolver{
		CanBindFn: func(p models.Principal, _ uuid.UUID, refs []models.SecretRef) error {
			gotPrincipal, gotRefs = p, refs
			return nil
		},
	}
	svc := New(db, &mocks.MockCache{}, &mocks.MockExecuter{}, resolver)

	err := svc.UpdatePostIt(member, id, map[string]any{
		"params":   map[string]string{"$credential": "$MINE"},
		"bindings": map[string]models.SecretRef{"MINE": mine},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if gotPrincipal != member || len(gotRefs) != 1 || gotRefs[0] != mine {
		t.Errorf("CanBind(%v, %v), want (%v, [%v])", gotPrincipal, gotRefs, member, mine)
	}
	if gotSet["runas"] != member.ID {
		t.Errorf("runas = %v, want the editor", gotSet["runas"])
	}
}

func TestUpdatePostIt_KeepsBindingsWhenOnlyParamsChange(t *testing.T) {
	board := uuid.New()
	id := uuid.New()
	mine := models.SecretRef{Scope: models.MemberScope(board, member.ID), Name: "MINE"}

	db := boardOf(boardOwner, member)
	db.FindPostItFn = func(uuid.UUID) (*models.PostIts, error) {
		return &models.PostIts{Id: id, Board: board, RunAs: member.ID, Bindings: map[string]models.SecretRef{"MINE": mine}}, nil
	}
	var gotRefs []models.SecretRef
	resolver := &mocks.MockSecretResolver{
		CanBindFn: func(_ models.Principal, _ uuid.UUID, refs []models.SecretRef) error {
			gotRefs = refs
			return nil
		},
	}
	svc := New(db, &mocks.MockCache{}, &mocks.MockExecuter{}, resolver)

	if err := svc.UpdatePostIt(member, id, map[string]any{"params": map[string]string{"$base": "EUR"}}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(gotRefs) != 1 || gotRefs[0] != mine {
		t.Errorf("CanBind refs = %v, want the stored binding re-checked", gotRefs)
	}
}

func TestUpdatePostIt_RateOnlyDoesNotChangeWhoItRunsAs(t *testing.T) {
	board := uuid.New()
	id := uuid.New()

	db := boardOf(boardOwner, member)
	db.FindPostItFn = func(uuid.UUID) (*models.PostIts, error) {
		return &models.PostIts{Id: id, Board: board, RunAs: boardOwner.ID}, nil
	}
	var gotSet map[string]any
	db.UpdatePostItFn = func(_ uuid.UUID, set map[string]any) error {
		gotSet = set
		return nil
	}
	svc := newService(db, nil, nil)

	if err := svc.UpdatePostIt(member, id, map[string]any{"rate": 5}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if _, touched := gotSet["runas"]; touched {
		t.Error("editing the rate rebound the card")
	}
}

func TestUpdatePostIt_RequiresMembership(t *testing.T) {
	db := boardOf(boardOwner)
	db.FindPostItFn = func(id uuid.UUID) (*models.PostIts, error) {
		return &models.PostIts{Id: id, Board: uuid.New()}, nil
	}
	updated := false
	db.UpdatePostItFn = func(uuid.UUID, map[string]any) error {
		updated = true
		return nil
	}
	svc := newService(db, nil, nil)

	if err := svc.UpdatePostIt(outsider, uuid.New(), map[string]any{"rate": 5}); !errors.Is(err, ErrNotAMember) {
		t.Errorf("got %v, want ErrNotAMember", err)
	}
	if updated {
		t.Error("the post-it was updated anyway")
	}
}

func boundCard(board uuid.UUID, runAs string) *models.PostIts {
	return &models.PostIts{
		Id:       uuid.New(),
		Board:    board,
		RunAs:    runAs,
		Resource: &url.URL{Scheme: "https", Host: "example.com"},
		Params:   map[string]string{"$credential": "$MINE"},
		Request: models.Request{
			Method:  "GET",
			Headers: map[string]string{"apikey": "$credential", "X-Shared": "$SHARED"},
		},
		Bindings: map[string]models.SecretRef{"MINE": {Scope: models.MemberScope(board, member.ID), Name: "MINE"}},
	}
}

func TestExecutePostIt_ResolvesEveryTokenAsWhoTheCardRunsAs(t *testing.T) {
	board := uuid.New()
	postit := boundCard(board, member.ID)
	mine := postit.Bindings["MINE"]
	shared := models.SecretRef{Scope: models.BoardScope(board), Name: "SHARED"}

	var gotPrincipal models.Principal
	var gotBoard uuid.UUID
	var gotRefs []models.SecretRef
	resolver := &mocks.MockSecretResolver{
		ResolveAsFn: func(p models.Principal, b uuid.UUID, refs []models.SecretRef) (map[models.SecretRef]string, error) {
			gotPrincipal, gotBoard, gotRefs = p, b, refs
			return map[models.SecretRef]string{mine: "member-value", shared: "board-value"}, nil
		},
	}
	var executed *models.PostIts
	run := &mocks.MockExecuter{
		ExecuteFn: func(p *models.PostIts) (any, error) {
			executed = p
			return map[string]any{}, nil
		},
	}
	svc := New(boardOf(boardOwner, member), &mocks.MockCache{}, run, resolver)

	if _, err := svc.ExecutePostIt(postit); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if gotPrincipal.ID != member.ID || gotBoard != board {
		t.Errorf("resolved as (%v, %v), want (%v, %v)", gotPrincipal, gotBoard, member, board)
	}
	if len(gotRefs) != 2 {
		t.Errorf("refs = %v, want the explicit binding and the board default", gotRefs)
	}
	if executed.Params["$credential"] != "member-value" || executed.Params["$SHARED"] != "board-value" {
		t.Errorf("params = %v", executed.Params)
	}
}

func TestExecutePostIt_CardsWithoutRunAsRunAsTheBoardOwner(t *testing.T) {
	board := uuid.New()
	postit := boundCard(board, "")

	var gotPrincipal models.Principal
	resolver := &mocks.MockSecretResolver{
		ResolveAsFn: func(p models.Principal, _ uuid.UUID, _ []models.SecretRef) (map[models.SecretRef]string, error) {
			gotPrincipal = p
			return map[models.SecretRef]string{postit.Bindings["MINE"]: "v"}, nil
		},
	}
	svc := New(boardOf(boardOwner, member), &mocks.MockCache{}, &mocks.MockExecuter{}, resolver)

	if _, err := svc.ExecutePostIt(postit); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if gotPrincipal.ID != boardOwner.ID {
		t.Errorf("resolved as %q, want the board owner", gotPrincipal.ID)
	}
}

// A binding was allowed when the card was saved; if that is no longer true
// (access revoked, secret deleted) the card must stop rather than send the
// token verbatim to the provider.
func TestExecutePostIt_UnavailableBindingStopsTheCard(t *testing.T) {
	board := uuid.New()

	cases := []struct {
		label    string
		resolver *mocks.MockSecretResolver
	}{
		{"revoked", &mocks.MockSecretResolver{
			ResolveAsFn: func(models.Principal, uuid.UUID, []models.SecretRef) (map[models.SecretRef]string, error) {
				return nil, infrastructure.ErrForbidden
			},
		}},
		{"deleted", &mocks.MockSecretResolver{
			ResolveAsFn: func(models.Principal, uuid.UUID, []models.SecretRef) (map[models.SecretRef]string, error) {
				return map[models.SecretRef]string{}, nil
			},
		}},
	}

	for _, c := range cases {
		ran := false
		run := &mocks.MockExecuter{ExecuteFn: func(*models.PostIts) (any, error) {
			ran = true
			return nil, nil
		}}
		svc := New(boardOf(boardOwner, member), &mocks.MockCache{}, run, c.resolver)

		_, err := svc.ExecutePostIt(boundCard(board, member.ID))
		if !errors.Is(err, ErrCredentialUnavailable) {
			t.Errorf("%s: got %v, want ErrCredentialUnavailable", c.label, err)
		}
		if ran {
			t.Errorf("%s: the request went out anyway", c.label)
		}
	}
}
