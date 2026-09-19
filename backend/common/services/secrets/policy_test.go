package secrets

import (
	"errors"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

func testPolicy() *policy {
	return NewPolicy(&mocks.MockDB{
		FindBoardFn: func(id uuid.UUID) (*models.Board, error) {
			return &models.Board{Id: id, Owner: owner.ID, Collaborators: []string{collaborator.ID}}, nil
		},
	})
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
	})

	if err := p.CanView(owner, models.BoardScope(uuid.New())); !errors.Is(err, ErrForbidden) {
		t.Errorf("got %v, want ErrForbidden", err)
	}
}
