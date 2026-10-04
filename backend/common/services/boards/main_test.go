package boards

import (
	"errors"
	"slices"
	"testing"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/services/access"
	"github.com/google/uuid"
)

// The board below is owned by owner, edited by ana and viewed by bob; eve is
// a stranger.
var (
	ownerUser = models.User{Id: uuid.New(), Name: "Owner", Email: "owner@example.com"}
	anaUser   = models.User{Id: uuid.New(), Name: "Ana", Email: "ana@example.com"}
	bobUser   = models.User{Id: uuid.New(), Name: "Bob", Email: "bob@example.com"}
	eveUser   = models.User{Id: uuid.New(), Name: "Eve", Email: "eve@example.com"}

	owner    = ownerUser.Principal()
	ana      = anaUser.Principal()
	bob      = bobUser.Principal()
	outsider = eveUser.Principal()
)

func boardFixture(id uuid.UUID) *models.Board {
	return &models.Board{Id: id, Owner: owner.ID, Members: []models.BoardMember{
		{User: ana.ID, Role: models.BoardEditor},
		{User: bob.ID, Role: models.BoardViewer},
	}}
}

// boardDB serves the board above under id, and nothing else.
func boardDB(id uuid.UUID) *mocks.MockDB {
	return &mocks.MockDB{
		FindBoardFn: func(got uuid.UUID) (*models.Board, error) {
			if got != id {
				return nil, errors.New("no documents")
			}
			return boardFixture(id), nil
		},
	}
}

func users() *mocks.MemoryUserStore {
	return &mocks.MemoryUserStore{Users: []models.User{ownerUser, anaUser, bobUser, eveUser}}
}

func newService(db *mocks.MockDB, purger infrastructure.ScopePurger) *BoardService {
	return New(db, purger, access.New(), users())
}

func TestCreateBoard_TheCallerOwnsIt(t *testing.T) {
	var gotName, gotOwner string
	db := &mocks.MockDB{
		CreateBoardFn: func(name, owner string) (*models.Board, error) {
			gotName, gotOwner = name, owner
			return &models.Board{Id: uuid.New(), Name: name, Owner: owner}, nil
		},
	}

	board, err := newService(db, &purgeRecorder{}).CreateBoard(ana, "My Board")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotName != "My Board" || gotOwner != ana.ID {
		t.Errorf("delegated (%q,%q), want (My Board, ana)", gotName, gotOwner)
	}
	if board.Role != models.BoardOwner {
		t.Errorf("role = %q, want owner", board.Role)
	}
}

func TestCreateBoard_AnonymousIsForbidden(t *testing.T) {
	if _, err := newService(&mocks.MockDB{}, &purgeRecorder{}).CreateBoard(models.Principal{}, "B"); !errors.Is(err, ErrForbidden) {
		t.Errorf("got %v, want ErrForbidden", err)
	}
}

// Each board in the listing says what the caller may do with it.
func TestGetUserBoards_CarriesTheCallersRole(t *testing.T) {
	id := uuid.New()
	db := &mocks.MockDB{
		FindUserBoardsFn: func(user string) ([]models.Board, error) {
			if user != bob.ID {
				t.Errorf("user = %q, want bob", user)
			}
			return []models.Board{*boardFixture(id)}, nil
		},
	}

	got, err := newService(db, &purgeRecorder{}).GetUserBoards(bob)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Role != models.BoardViewer {
		t.Errorf("got %+v, want the board as viewer", got)
	}
}

func TestGetUserBoards_AnonymousIsForbidden(t *testing.T) {
	if _, err := newService(&mocks.MockDB{}, &purgeRecorder{}).GetUserBoards(models.Principal{}); !errors.Is(err, ErrForbidden) {
		t.Errorf("got %v, want ErrForbidden", err)
	}
}

// Who may do what, for every route of the service.
func TestRoles_Matrix(t *testing.T) {
	id := uuid.New()
	db := boardDB(id)
	db.FindBoardPostItsFn = func(uuid.UUID) ([]models.PostIts, error) { return []models.PostIts{}, nil }
	db.ConnectPostItsFn = func(b, s, t uuid.UUID) (*models.Strand, error) { return &models.Strand{}, nil }
	db.DisconnectPostItsFn = func(b, s uuid.UUID) error { return nil }
	db.UpdateBoardNameFn = func(uuid.UUID, string) error { return nil }
	db.DeleteBoardFn = func(uuid.UUID) error { return nil }
	db.SetBoardMemberFn = func(uuid.UUID, string, models.BoardRole) error { return nil }
	db.RemoveBoardMemberFn = func(uuid.UUID, string) error { return nil }
	svc := newService(db, &purgeRecorder{})

	calls := []struct {
		name string
		min  models.BoardRole
		call func(models.Principal) error
	}{
		{"get", models.BoardViewer, func(p models.Principal) error { _, err := svc.GetBoard(p, id); return err }},
		{"post-its", models.BoardViewer, func(p models.Principal) error { _, err := svc.GetBoardPostIts(p, id); return err }},
		{"members", models.BoardViewer, func(p models.Principal) error { _, err := svc.Members(p, id); return err }},
		{"connect", models.BoardEditor, func(p models.Principal) error {
			_, err := svc.ConnectPostIts(p, id, uuid.New(), uuid.New())
			return err
		}},
		{"disconnect", models.BoardEditor, func(p models.Principal) error { return svc.DisconnectPostIts(p, id, uuid.New()) }},
		{"rename", models.BoardOwner, func(p models.Principal) error { return svc.UpdateBoardName(p, id, "Renamed") }},
		{"set member", models.BoardOwner, func(p models.Principal) error {
			_, err := svc.SetMember(p, id, eveUser.Email, models.BoardViewer)
			return err
		}},
		{"remove someone else", models.BoardOwner, func(p models.Principal) error {
			return svc.RemoveMember(p, id, eveUser.Id.String())
		}},
		{"delete", models.BoardOwner, func(p models.Principal) error { return svc.DeleteBoard(p, id) }},
	}

	roles := []struct {
		principal models.Principal
		role      models.BoardRole
	}{{owner, models.BoardOwner}, {ana, models.BoardEditor}, {bob, models.BoardViewer}, {outsider, ""}}

	for _, c := range calls {
		for _, r := range roles {
			err := c.call(r.principal)
			allowed := r.role.AtLeast(c.min)
			if allowed && err != nil {
				t.Errorf("%s as %s: %v", c.name, r.role, err)
			}
			if !allowed && !errors.Is(err, ErrForbidden) {
				t.Errorf("%s as %q: got %v, want ErrForbidden", c.name, r.role, err)
			}
		}
	}

	if _, err := svc.GetBoard(owner, uuid.New()); !errors.Is(err, ErrForbidden) {
		t.Errorf("missing board: got %v, want ErrForbidden", err)
	}
}

func TestGetBoard_CarriesTheCallersRole(t *testing.T) {
	id := uuid.New()
	board, err := newService(boardDB(id), &purgeRecorder{}).GetBoard(ana, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if board.Role != models.BoardEditor {
		t.Errorf("role = %q, want editor", board.Role)
	}
}

// Members are listed with who they are, the owner first.
func TestMembers_ListsSummariesOwnerFirst(t *testing.T) {
	id := uuid.New()
	members, err := newService(boardDB(id), &purgeRecorder{}).Members(bob, id)
	if err != nil {
		t.Fatalf("members: %v", err)
	}

	want := []models.BoardMemberSummary{
		{User: ownerUser.Summary(), Role: models.BoardOwner},
		{User: anaUser.Summary(), Role: models.BoardEditor},
		{User: bobUser.Summary(), Role: models.BoardViewer},
	}
	if !slices.Equal(members, want) {
		t.Errorf("members = %+v, want %+v", members, want)
	}
}

// The owner shares by email; the address is matched however it was typed.
func TestSetMember_ByEmail(t *testing.T) {
	id := uuid.New()
	var gotUser string
	var gotRole models.BoardRole
	db := boardDB(id)
	db.SetBoardMemberFn = func(_ uuid.UUID, user string, role models.BoardRole) error {
		gotUser, gotRole = user, role
		return nil
	}

	member, err := newService(db, &purgeRecorder{}).SetMember(owner, id, "  EVE@example.com ", models.BoardEditor)
	if err != nil {
		t.Fatalf("set: %v", err)
	}
	if gotUser != outsider.ID || gotRole != models.BoardEditor {
		t.Errorf("stored (%q, %q), want (eve, editor)", gotUser, gotRole)
	}
	if member.User != eveUser.Summary() || member.Role != models.BoardEditor {
		t.Errorf("member = %+v", member)
	}
}

func TestSetMember_Refusals(t *testing.T) {
	id := uuid.New()
	db := boardDB(id)
	db.SetBoardMemberFn = func(uuid.UUID, string, models.BoardRole) error {
		t.Error("a refused member was stored")
		return nil
	}
	svc := newService(db, &purgeRecorder{})

	cases := []struct {
		name  string
		email string
		role  models.BoardRole
		want  error
	}{
		{"unknown email", "nobody@example.com", models.BoardViewer, infrastructure.ErrUserNotFound},
		{"owner role", eveUser.Email, models.BoardOwner, ErrInvalidRole},
		{"made-up role", eveUser.Email, "admin", ErrInvalidRole},
		{"the owner themselves", ownerUser.Email, models.BoardViewer, ErrOwnerRole},
	}
	for _, c := range cases {
		if _, err := svc.SetMember(owner, id, c.email, c.role); !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, err, c.want)
		}
	}
}

// Anyone may leave a board on their own.
func TestRemoveMember_AMemberMayLeave(t *testing.T) {
	id := uuid.New()
	removed := ""
	db := boardDB(id)
	db.RemoveBoardMemberFn = func(_ uuid.UUID, user string) error {
		removed = user
		return nil
	}

	if err := newService(db, &purgeRecorder{}).RemoveMember(bob, id, bob.ID); err != nil {
		t.Fatalf("leave: %v", err)
	}
	if removed != bob.ID {
		t.Errorf("removed %q, want bob", removed)
	}
}

// The owner is not a member: "removing" them would only destroy what they
// keep on the board.
func TestRemoveMember_TheOwnerCannotBeRemoved(t *testing.T) {
	id := uuid.New()
	db := boardDB(id)
	db.RemoveBoardMemberFn = func(uuid.UUID, string) error {
		t.Error("the owner was removed")
		return nil
	}
	purger := &purgeRecorder{}

	if err := newService(db, purger).RemoveMember(owner, id, owner.ID); !errors.Is(err, ErrOwnerRole) {
		t.Errorf("got %v, want ErrOwnerRole", err)
	}
	if len(purger.purged) != 0 {
		t.Errorf("purged %v", purger.purged)
	}
}

type purgeRecorder struct {
	purged []models.SecretScope
}

func (p *purgeRecorder) Purge(scope models.SecretScope) error {
	p.purged = append(p.purged, scope)
	return nil
}

// Deleting a board must also destroy its secrets and the key that encrypted
// them, or a dump taken later could still be decrypted. What each member kept
// there for themselves goes with it.
func TestDeleteBoard_PurgesItsSecrets(t *testing.T) {
	id := uuid.New()
	deleted := false
	db := boardDB(id)
	db.DeleteBoardFn = func(got uuid.UUID) error {
		deleted = got == id
		return nil
	}
	purger := &purgeRecorder{}

	if err := newService(db, purger).DeleteBoard(owner, id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if !deleted {
		t.Error("the board itself was not deleted")
	}

	want := []models.SecretScope{
		models.BoardScope(id),
		models.MemberScope(id, owner.ID),
		models.MemberScope(id, ana.ID),
		models.MemberScope(id, bob.ID),
	}
	if len(purger.purged) != len(want) {
		t.Fatalf("purged %v, want %v", purger.purged, want)
	}
	for _, scope := range want {
		if !slices.Contains(purger.purged, scope) {
			t.Errorf("%s was not purged", scope.Key())
		}
	}
}

// A member who leaves takes nothing with them: what they kept on the board is
// destroyed, not left for the day they are added again.
func TestRemoveMember_PurgesWhatTheyKeptOnTheBoard(t *testing.T) {
	id := uuid.New()
	db := boardDB(id)
	purger := &purgeRecorder{}

	if err := newService(db, purger).RemoveMember(owner, id, ana.ID); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if len(purger.purged) != 1 || purger.purged[0] != models.MemberScope(id, ana.ID) {
		t.Errorf("purged %v, want ana's member scope", purger.purged)
	}
}
