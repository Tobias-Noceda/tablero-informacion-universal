package secrets

import (
	"errors"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/services/access"
	"github.com/google/uuid"
)

var (
	opsGroup = models.Group{Id: uuid.New(), Name: "ops", Owner: owner.ID, Members: []string{collaborator.ID}}
	viewer   = models.Principal{ID: "viewer-id"}
	alice    = models.Principal{ID: "alice"}
)

// testPolicy serves every board as owned by owner, edited by collaborator and
// alice, and viewed by viewer.
func testPolicy() *policy {
	return NewPolicy(&mocks.MockDB{
		FindBoardFn: func(id uuid.UUID) (*models.Board, error) {
			return &models.Board{Id: id, Owner: owner.ID, Members: []models.BoardMember{
				{User: collaborator.ID, Role: models.BoardEditor},
				{User: alice.ID, Role: models.BoardEditor},
				{User: viewer.ID, Role: models.BoardViewer},
			}}, nil
		},
	}, &mocks.MemoryGroupStore{Groups: []models.Group{opsGroup}}, access.New())
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
		{viewer, false, false},
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

// The platform's own credentials are for platform admins, and nobody else,
// whatever boards they own.
func TestPolicy_SystemIsForAdminsOnly(t *testing.T) {
	p := testPolicy()

	if err := p.CanManage(admin, models.SystemScope); err != nil {
		t.Errorf("admin: CanManage = %v", err)
	}
	if err := p.CanView(admin, models.SystemScope); err != nil {
		t.Errorf("admin: CanView = %v", err)
	}

	for _, caller := range []models.Principal{owner, stranger, anonymous} {
		if err := p.CanManage(caller, models.SystemScope); !errors.Is(err, ErrForbidden) {
			t.Errorf("caller %q: CanManage = %v, want ErrForbidden", caller.ID, err)
		}
		if err := p.CanView(caller, models.SystemScope); !errors.Is(err, ErrForbidden) {
			t.Errorf("caller %q: CanView = %v, want ErrForbidden", caller.ID, err)
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
	}, &mocks.MemoryGroupStore{}, access.New())

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

	// What one keeps on a board is for binding into cards, which a viewer
	// never does.
	if err := p.CanView(viewer, models.MemberScope(board, viewer.ID)); !errors.Is(err, ErrForbidden) {
		t.Errorf("a viewer got a member scope: %v", err)
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
		{"user secret by its user", collaborator, board, models.UserScope(collaborator.ID), true},
		{"user secret by someone else", owner, board, models.UserScope(collaborator.ID), false},
		{"user secret by its user on a board they are not on", stranger, board, models.UserScope(stranger.ID), false},
		{"board secret by a viewer", viewer, board, models.BoardScope(board), false},
		{"user secret by a viewer", viewer, board, models.UserScope(viewer.ID), false},
		{"member secret by a viewer", viewer, board, models.MemberScope(board, viewer.ID), false},
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
	}, &mocks.MemoryGroupStore{}, access.New())
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

func TestPolicy_CanUse_Grants(t *testing.T) {
	p := testPolicy()
	board := uuid.New()
	otherBoard := uuid.New()
	alicesKey := models.UserScope("alice")

	grant := func(to models.Audience, restrictedTo string) *models.Secret {
		return &models.Secret{Scope: alicesKey, Name: "KEY", Grants: []models.Grant{{To: to, Board: restrictedTo}}}
	}
	user := func(id string) models.Audience { return models.Audience{Kind: models.AudienceUser, ID: id} }

	cases := []struct {
		label  string
		caller models.Principal
		board  uuid.UUID
		secret *models.Secret
		want   bool
	}{
		{"granted user anywhere", collaborator, otherBoard, grant(user(collaborator.ID), ""), true},
		{"granted user on that board", collaborator, board, grant(user(collaborator.ID), board.String()), true},
		{"granted user on another board", collaborator, otherBoard, grant(user(collaborator.ID), board.String()), false},
		{"someone else", stranger, board, grant(user(collaborator.ID), ""), false},
		{"granted group member", collaborator, board, grant(models.Audience{Kind: models.AudienceGroup, ID: opsGroup.Id.String()}, ""), true},
		{"granted group outsider", stranger, board, grant(models.Audience{Kind: models.AudienceGroup, ID: opsGroup.Id.String()}, ""), false},
		{"granted board members, a member", collaborator, board, grant(models.Audience{Kind: models.AudienceBoard, ID: board.String()}, ""), true},
		{"granted board members, from another board", collaborator, otherBoard, grant(models.Audience{Kind: models.AudienceBoard, ID: board.String()}, ""), false},
		{"granted board members, a stranger", stranger, board, grant(models.Audience{Kind: models.AudienceBoard, ID: board.String()}, ""), false},
		{"granted board members, a viewer", viewer, board, grant(models.Audience{Kind: models.AudienceBoard, ID: board.String()}, ""), false},
		{"granted user, who only views the board", viewer, board, grant(user(viewer.ID), ""), false},
		{"the owner still can", alice, board, grant(user(collaborator.ID), ""), true},
		{"no grants", collaborator, board, &models.Secret{Scope: alicesKey, Name: "KEY"}, false},
		{"a grant on a system secret is ignored", collaborator, board, &models.Secret{Scope: models.SystemScope, Name: "KEY", Grants: []models.Grant{{To: user(collaborator.ID)}}}, false},
		{"anonymous", anonymous, board, grant(user(""), ""), false},
	}

	for _, c := range cases {
		err := p.CanUse(c.caller, c.board, c.secret)
		if got := err == nil; got != c.want {
			t.Errorf("%s: CanUse = %v, want allowed=%v", c.label, err, c.want)
		}
	}
}
