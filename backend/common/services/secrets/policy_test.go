package secrets

import (
	"errors"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

var opsGroup = models.Group{Id: uuid.New(), Name: "ops", Owner: owner.ID, Members: []string{collaborator.ID}}

func testPolicy() *policy {
	return NewPolicy(&mocks.MockDB{
		FindBoardFn: func(id uuid.UUID) (*models.Board, error) {
			return &models.Board{Id: id, Owner: owner.ID, Collaborators: []string{collaborator.ID}}, nil
		},
	}, &mocks.MemoryGroupStore{Groups: []models.Group{opsGroup}})
}

func TestPolicy_Group(t *testing.T) {
	p := testPolicy()
	scope := models.GroupScope(opsGroup.Id)

	cases := []struct {
		caller    models.Principal
		canManage bool
		canView   bool
	}{
		{owner, true, true},
		{collaborator, false, true},
		{stranger, false, false},
		{anonymous, false, false},
	}

	for _, c := range cases {
		if got := p.CanManage(c.caller, scope) == nil; got != c.canManage {
			t.Errorf("caller %q: CanManage = %v, want %v", c.caller.ID, got, c.canManage)
		}
		if got := p.CanView(c.caller, scope) == nil; got != c.canView {
			t.Errorf("caller %q: CanView = %v, want %v", c.caller.ID, got, c.canView)
		}
	}

	if err := p.CanView(owner, models.GroupScope(uuid.New())); !errors.Is(err, ErrForbidden) {
		t.Errorf("an unknown group answered %v, want ErrForbidden", err)
	}
}

func TestPolicy_Board(t *testing.T) {
	p := testPolicy()
	scope := models.BoardScope(uuid.New())

	cases := []struct {
		caller    models.Principal
		canManage bool
		canView   bool
	}{
		{owner, true, true},
		{collaborator, false, true},
		{stranger, false, false},
		{anonymous, false, false},
	}

	for _, c := range cases {
		if got := p.CanManage(c.caller, scope) == nil; got != c.canManage {
			t.Errorf("caller %q: CanManage = %v, want %v", c.caller.ID, got, c.canManage)
		}
		if got := p.CanView(c.caller, scope) == nil; got != c.canView {
			t.Errorf("caller %q: CanView = %v, want %v", c.caller.ID, got, c.canView)
		}
	}
}

func TestPolicy_User(t *testing.T) {
	p := testPolicy()
	scope := models.UserScope("alice")

	if err := p.CanManage(models.Principal{ID: "alice"}, scope); err != nil {
		t.Errorf("a user cannot manage their own secrets: %v", err)
	}
	if err := p.CanView(models.Principal{ID: "alice"}, scope); err != nil {
		t.Errorf("a user cannot view their own secrets: %v", err)
	}

	for _, caller := range []string{"bob", ""} {
		if err := p.CanManage(models.Principal{ID: caller}, scope); !errors.Is(err, ErrForbidden) {
			t.Errorf("caller %q managed another user's secrets: %v", caller, err)
		}
		if err := p.CanView(models.Principal{ID: caller}, scope); !errors.Is(err, ErrForbidden) {
			t.Errorf("caller %q viewed another user's secrets: %v", caller, err)
		}
	}
}

// Nobody can be told apart until authentication exists, so the system scope
// is open. This test pins that down so the day it changes is a deliberate one.
func TestPolicy_SystemIsOpenUntilAuthExists(t *testing.T) {
	p := testPolicy()

	for _, caller := range []models.Principal{owner, stranger, anonymous} {
		if err := p.CanManage(caller, models.SystemScope); err != nil {
			t.Errorf("caller %q: CanManage = %v", caller.ID, err)
		}
		if err := p.CanView(caller, models.SystemScope); err != nil {
			t.Errorf("caller %q: CanView = %v", caller.ID, err)
		}
	}
}

// A malformed scope must never be treated as an allowed one.
func TestPolicy_RejectsInvalidScopes(t *testing.T) {
	p := testPolicy()
	principal := owner

	for _, scope := range []models.SecretScope{
		{Kind: "team", Owner: "x"},
		{Kind: models.ScopeBoard, Owner: "not-a-uuid"},
		{Kind: models.ScopeSystem, Owner: "x"},
	} {
		if err := p.CanManage(principal, scope); !errors.Is(err, ErrForbidden) {
			t.Errorf("%+v: CanManage = %v, want ErrForbidden", scope, err)
		}
		if err := p.CanView(principal, scope); !errors.Is(err, ErrForbidden) {
			t.Errorf("%+v: CanView = %v, want ErrForbidden", scope, err)
		}
	}
}

// A board the store cannot find is indistinguishable from one the caller may
// not touch.
func TestPolicy_UnknownBoardIsForbidden(t *testing.T) {
	p := NewPolicy(&mocks.MockDB{
		FindBoardFn: func(uuid.UUID) (*models.Board, error) {
			return nil, errors.New("no documents")
		},
	}, &mocks.MemoryGroupStore{})

	if err := p.CanView(owner, models.BoardScope(uuid.New())); !errors.Is(err, ErrForbidden) {
		t.Errorf("got %v, want ErrForbidden", err)
	}
}

func TestPolicy_Member(t *testing.T) {
	p := testPolicy()
	board := uuid.New()
	scope := models.MemberScope(board, collaborator.ID)

	if err := p.CanManage(collaborator, scope); err != nil {
		t.Errorf("a member cannot manage their own board credentials: %v", err)
	}
	if err := p.CanView(collaborator, scope); err != nil {
		t.Errorf("a member cannot view their own board credentials: %v", err)
	}

	for _, caller := range []models.Principal{owner, stranger, anonymous} {
		if err := p.CanManage(caller, scope); !errors.Is(err, ErrForbidden) {
			t.Errorf("caller %q managed another member's credentials: %v", caller.ID, err)
		}
		if err := p.CanView(caller, scope); !errors.Is(err, ErrForbidden) {
			t.Errorf("caller %q viewed another member's credentials: %v", caller.ID, err)
		}
	}

	if err := p.CanManage(stranger, models.MemberScope(board, stranger.ID)); !errors.Is(err, ErrForbidden) {
		t.Errorf("someone who is not on the board got a member scope there: %v", err)
	}
}

func TestPolicy_CanUse(t *testing.T) {
	p := testPolicy()
	board := uuid.New()
	otherBoard := uuid.New()

	cases := []struct {
		label  string
		caller models.Principal
		board  uuid.UUID
		scope  models.SecretScope
		want   bool
	}{
		{"board secret by owner", owner, board, models.BoardScope(board), true},
		{"board secret by collaborator", collaborator, board, models.BoardScope(board), true},
		{"board secret by stranger", stranger, board, models.BoardScope(board), false},
		{"board secret from another board", owner, otherBoard, models.BoardScope(board), false},
		{"member secret by its user", collaborator, board, models.MemberScope(board, collaborator.ID), true},
		{"member secret by the board owner", owner, board, models.MemberScope(board, collaborator.ID), false},
		{"member secret on another board", collaborator, otherBoard, models.MemberScope(board, collaborator.ID), false},
		{"user secret by its user", stranger, board, models.UserScope(stranger.ID), true},
		{"user secret by someone else", owner, board, models.UserScope(stranger.ID), false},
		{"group secret by a group member on any board", collaborator, otherBoard, models.GroupScope(opsGroup.Id), true},
		{"group secret by the group owner", owner, board, models.GroupScope(opsGroup.Id), true},
		{"group secret by an outsider", stranger, board, models.GroupScope(opsGroup.Id), false},
		{"secret of an unknown group", owner, board, models.GroupScope(uuid.New()), false},
		{"system secret", owner, board, models.SystemScope, false},
		{"anonymous", anonymous, board, models.BoardScope(board), false},
		{"invalid scope", owner, board, models.SecretScope{Kind: "team", Owner: "x"}, false},
	}

	for _, c := range cases {
		err := p.CanUse(c.caller, c.board, &models.Secret{Scope: c.scope, Name: "KEY"})
		if got := err == nil; got != c.want {
			t.Errorf("%s: CanUse = %v, want allowed=%v", c.label, err, c.want)
		}
		if err != nil && !errors.Is(err, ErrForbidden) {
			t.Errorf("%s: got %v, want ErrForbidden", c.label, err)
		}
	}
}

// Leaving the board revokes what the membership granted, even for a secret
// that lives in the member scope itself.
func TestPolicy_CanUse_RemovedCollaboratorLosesAccess(t *testing.T) {
	p := NewPolicy(&mocks.MockDB{
		FindBoardFn: func(id uuid.UUID) (*models.Board, error) {
			return &models.Board{Id: id, Owner: owner.ID}, nil
		},
	}, &mocks.MemoryGroupStore{})
	board := uuid.New()

	for _, scope := range []models.SecretScope{
		models.BoardScope(board),
		models.MemberScope(board, collaborator.ID),
	} {
		if err := p.CanUse(collaborator, board, &models.Secret{Scope: scope, Name: "KEY"}); !errors.Is(err, ErrForbidden) {
			t.Errorf("%s: got %v, want ErrForbidden", scope.Key(), err)
		}
	}
}
