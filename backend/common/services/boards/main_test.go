package boards

import (
	"errors"
	"slices"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

var (
	owner    = models.Principal{ID: "owner"}
	ana      = models.Principal{ID: "ana"}
	outsider = models.Principal{ID: "eve"}
)

// boardDB serves one board owned by "owner" with "ana" as a collaborator.
func boardDB(id uuid.UUID) *mocks.MockDB {
	return &mocks.MockDB{
		FindBoardFn: func(got uuid.UUID) (*models.Board, error) {
			if got != id {
				return nil, errors.New("no documents")
			}
			return &models.Board{Id: id, Owner: "owner", Collaborators: []string{"ana"}}, nil
		},
	}
}

func TestCreateBoard_TheCallerOwnsIt(t *testing.T) {
	var gotName, gotOwner string
	db := &mocks.MockDB{
		CreateBoardFn: func(name, owner string) (*models.Board, error) {
			gotName, gotOwner = name, owner
			return &models.Board{Id: uuid.New(), Name: name, Owner: owner}, nil
		},
	}

	board, err := New(db, &purgeRecorder{}).CreateBoard(ana, "My Board")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotName != "My Board" || gotOwner != "ana" {
		t.Errorf("delegated (%q,%q), want (My Board, ana)", gotName, gotOwner)
	}
	if board.Name != "My Board" {
		t.Errorf("board name = %q", board.Name)
	}
}

func TestCreateBoard_AnonymousIsForbidden(t *testing.T) {
	if _, err := New(&mocks.MockDB{}, &purgeRecorder{}).CreateBoard(models.Principal{}, "B"); !errors.Is(err, ErrForbidden) {
		t.Errorf("got %v, want ErrForbidden", err)
	}
}

func TestGetUserBoards_ListsTheCallersBoards(t *testing.T) {
	want := []models.Board{{Id: uuid.New()}, {Id: uuid.New()}}
	db := &mocks.MockDB{
		FindUserBoardsFn: func(user string) ([]models.Board, error) {
			if user != "ana" {
				t.Errorf("user = %q, want ana", user)
			}
			return want, nil
		},
	}

	got, err := New(db, &purgeRecorder{}).GetUserBoards(ana)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("got %d boards, want 2", len(got))
	}
}

func TestGetUserBoards_AnonymousIsForbidden(t *testing.T) {
	if _, err := New(&mocks.MockDB{}, &purgeRecorder{}).GetUserBoards(models.Principal{}); !errors.Is(err, ErrForbidden) {
		t.Errorf("got %v, want ErrForbidden", err)
	}
}

// Reading and drawing on a board is for its members; anyone else cannot tell
// it from a board that does not exist.
func TestMemberRoutes_AdmitMembersOnly(t *testing.T) {
	id := uuid.New()
	db := boardDB(id)
	db.FindBoardPostItsFn = func(uuid.UUID) ([]models.PostIts, error) { return []models.PostIts{{}}, nil }
	db.ConnectPostItsFn = func(b, s, t uuid.UUID) (*models.Strand, error) { return &models.Strand{}, nil }
	db.DisconnectPostItsFn = func(b, s uuid.UUID) error { return nil }
	svc := New(db, &purgeRecorder{})

	calls := map[string]func(models.Principal, uuid.UUID) error{
		"get": func(p models.Principal, b uuid.UUID) error {
			_, err := svc.GetBoard(p, b)
			return err
		},
		"post-its": func(p models.Principal, b uuid.UUID) error {
			_, err := svc.GetBoardPostIts(p, b)
			return err
		},
		"connect": func(p models.Principal, b uuid.UUID) error {
			_, err := svc.ConnectPostIts(p, b, uuid.New(), uuid.New())
			return err
		},
		"disconnect": func(p models.Principal, b uuid.UUID) error {
			return svc.DisconnectPostIts(p, b, uuid.New())
		},
	}

	for name, call := range calls {
		for _, member := range []models.Principal{owner, ana} {
			if err := call(member, id); err != nil {
				t.Errorf("%s as %s: %v", name, member.ID, err)
			}
		}
		if err := call(outsider, id); !errors.Is(err, ErrForbidden) {
			t.Errorf("%s as an outsider: got %v, want ErrForbidden", name, err)
		}
		if err := call(owner, uuid.New()); !errors.Is(err, ErrForbidden) {
			t.Errorf("%s on a missing board: got %v, want ErrForbidden", name, err)
		}
	}
}

// Renaming, deleting and choosing the collaborators is the owner's call.
func TestOwnerRoutes_AdmitTheOwnerOnly(t *testing.T) {
	id := uuid.New()
	db := boardDB(id)
	db.UpdateBoardNameFn = func(uuid.UUID, string) error { return nil }
	db.DeleteBoardFn = func(uuid.UUID) error { return nil }
	db.AddCollaboratorToBoardFn = func(uuid.UUID, string) error { return nil }
	db.RemoveCollaboratorFromBoardFn = func(uuid.UUID, string) error { return nil }
	svc := New(db, &purgeRecorder{})

	calls := map[string]func(models.Principal) error{
		"rename": func(p models.Principal) error { return svc.UpdateBoardName(p, id, "Renamed") },
		"add":    func(p models.Principal) error { return svc.AddCollaboratorToBoard(p, id, "bob") },
		"remove": func(p models.Principal) error { return svc.RemoveCollaboratorFromBoard(p, id, "bob") },
		"delete": func(p models.Principal) error { return svc.DeleteBoard(p, id) },
	}

	for name, call := range calls {
		for _, intruder := range []models.Principal{ana, outsider} {
			if err := call(intruder); !errors.Is(err, ErrForbidden) {
				t.Errorf("%s as %s: got %v, want ErrForbidden", name, intruder.ID, err)
			}
		}
		if err := call(owner); err != nil {
			t.Errorf("%s as the owner: %v", name, err)
		}
	}
}

func TestUpdateBoardName_Delegates(t *testing.T) {
	id := uuid.New()
	var gotName string
	db := boardDB(id)
	db.UpdateBoardNameFn = func(_ uuid.UUID, name string) error {
		gotName = name
		return nil
	}

	if err := New(db, &purgeRecorder{}).UpdateBoardName(owner, id, "Renamed"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotName != "Renamed" {
		t.Errorf("name = %q, want Renamed", gotName)
	}
}

func TestConnectPostIts_Delegates(t *testing.T) {
	id, src, tgt := uuid.New(), uuid.New(), uuid.New()
	var gb, gs, gt uuid.UUID
	db := boardDB(id)
	db.ConnectPostItsFn = func(b, s, t uuid.UUID) (*models.Strand, error) {
		gb, gs, gt = b, s, t
		return &models.Strand{Id: uuid.New(), Source: s, Target: t}, nil
	}

	if _, err := New(db, &purgeRecorder{}).ConnectPostIts(ana, id, src, tgt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gb != id || gs != src || gt != tgt {
		t.Errorf("connected (%v,%v,%v), want (%v,%v,%v)", gb, gs, gt, id, src, tgt)
	}
}

// A collaborator may walk away from a board on their own.
func TestRemoveCollaborator_ACollaboratorMayLeave(t *testing.T) {
	id := uuid.New()
	removed := ""
	db := boardDB(id)
	db.RemoveCollaboratorFromBoardFn = func(_ uuid.UUID, user string) error {
		removed = user
		return nil
	}

	if err := New(db, &purgeRecorder{}).RemoveCollaboratorFromBoard(ana, id, "ana"); err != nil {
		t.Fatalf("leave: %v", err)
	}
	if removed != "ana" {
		t.Errorf("removed %q, want ana", removed)
	}
}

// The owner is not a collaborator: "removing" them would only destroy what
// they keep on the board.
func TestRemoveCollaborator_TheOwnerIsNotACollaborator(t *testing.T) {
	id := uuid.New()
	db := boardDB(id)
	db.RemoveCollaboratorFromBoardFn = func(uuid.UUID, string) error {
		t.Error("the owner was removed")
		return nil
	}
	purger := &purgeRecorder{}

	if err := New(db, purger).RemoveCollaboratorFromBoard(owner, id, "owner"); !errors.Is(err, ErrOwnerIsNotACollaborator) {
		t.Errorf("got %v, want ErrOwnerIsNotACollaborator", err)
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
	db := &mocks.MockDB{
		FindBoardFn: func(got uuid.UUID) (*models.Board, error) {
			return &models.Board{Id: got, Owner: "owner", Collaborators: []string{"ana", "bob"}}, nil
		},
		DeleteBoardFn: func(got uuid.UUID) error {
			deleted = got == id
			return nil
		},
	}
	purger := &purgeRecorder{}

	if err := New(db, purger).DeleteBoard(owner, id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if !deleted {
		t.Error("the board itself was not deleted")
	}

	want := []models.SecretScope{
		models.BoardScope(id),
		models.MemberScope(id, "owner"),
		models.MemberScope(id, "ana"),
		models.MemberScope(id, "bob"),
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

// A collaborator who leaves takes nothing with them: what they kept on the
// board is destroyed, not left for the next person with the same id.
func TestRemoveCollaborator_PurgesWhatTheyKeptOnTheBoard(t *testing.T) {
	id := uuid.New()
	removed := false
	db := boardDB(id)
	db.RemoveCollaboratorFromBoardFn = func(got uuid.UUID, user string) error {
		removed = got == id && user == "ana"
		return nil
	}
	purger := &purgeRecorder{}

	if err := New(db, purger).RemoveCollaboratorFromBoard(owner, id, "ana"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if !removed {
		t.Error("the collaborator was not removed")
	}
	if len(purger.purged) != 1 || purger.purged[0] != models.MemberScope(id, "ana") {
		t.Errorf("purged %v, want the member scope", purger.purged)
	}
}
