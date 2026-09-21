package boards

import (
	"slices"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

func TestCreateBoard_Delegates(t *testing.T) {
	var gotName, gotOwner string
	db := &mocks.MockDB{
		CreateBoardFn: func(name, owner string) (*models.Board, error) {
			gotName, gotOwner = name, owner
			return &models.Board{Id: uuid.New(), Name: name, Owner: owner}, nil
		},
	}

	svc := New(db, &purgeRecorder{})
	board, err := svc.CreateBoard("My Board", "owner-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotName != "My Board" || gotOwner != "owner-1" {
		t.Errorf("delegated (%q,%q), want (My Board, owner-1)", gotName, gotOwner)
	}
	if board.Name != "My Board" {
		t.Errorf("board name = %q", board.Name)
	}
}

func TestConnectPostIts_Delegates(t *testing.T) {
	board, src, tgt := uuid.New(), uuid.New(), uuid.New()
	var gb, gs, gt uuid.UUID
	db := &mocks.MockDB{
		ConnectPostItsFn: func(b, s, t uuid.UUID) (*models.Strand, error) {
			gb, gs, gt = b, s, t
			return &models.Strand{Id: uuid.New(), Source: s, Target: t}, nil
		},
	}

	svc := New(db, &purgeRecorder{})
	if _, err := svc.ConnectPostIts(board, src, tgt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gb != board || gs != src || gt != tgt {
		t.Errorf("connected (%v,%v,%v), want (%v,%v,%v)", gb, gs, gt, board, src, tgt)
	}
}

func TestDisconnectPostIts_Delegates(t *testing.T) {
	board, strand := uuid.New(), uuid.New()
	var gb, gs uuid.UUID
	db := &mocks.MockDB{
		DisconnectPostItsFn: func(b, s uuid.UUID) error {
			gb, gs = b, s
			return nil
		},
	}

	svc := New(db, &purgeRecorder{})
	if err := svc.DisconnectPostIts(board, strand); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gb != board || gs != strand {
		t.Errorf("disconnected (%v,%v), want (%v,%v)", gb, gs, board, strand)
	}
}

func TestGetUserBoards_PropagatesResult(t *testing.T) {
	want := []models.Board{{Id: uuid.New()}, {Id: uuid.New()}}
	db := &mocks.MockDB{
		FindUserBoardsFn: func(cognitoID string) ([]models.Board, error) {
			if cognitoID != "user-x" {
				t.Errorf("cognitoID = %q, want user-x", cognitoID)
			}
			return want, nil
		},
	}

	svc := New(db, &purgeRecorder{})
	got, err := svc.GetUserBoards("user-x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("got %d boards, want 2", len(got))
	}
}

func TestGetBoard_Delegates(t *testing.T) {
	id := uuid.New()
	var got uuid.UUID
	db := &mocks.MockDB{
		FindBoardFn: func(b uuid.UUID) (*models.Board, error) {
			got = b
			return &models.Board{Id: b}, nil
		},
	}

	board, err := New(db, &purgeRecorder{}).GetBoard(id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != id || board.Id != id {
		t.Errorf("GetBoard(%v) used %v / returned %v", id, got, board.Id)
	}
}

func TestGetBoardPostIts_Delegates(t *testing.T) {
	id := uuid.New()
	db := &mocks.MockDB{
		FindBoardPostItsFn: func(b uuid.UUID) ([]models.PostIts, error) {
			if b != id {
				t.Errorf("board = %v, want %v", b, id)
			}
			return []models.PostIts{{Id: uuid.New()}}, nil
		},
	}

	got, err := New(db, &purgeRecorder{}).GetBoardPostIts(id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("got %d post-its, want 1", len(got))
	}
}

func TestDeleteBoard_Delegates(t *testing.T) {
	id := uuid.New()
	called := false
	db := &mocks.MockDB{
		FindBoardFn: func(got uuid.UUID) (*models.Board, error) {
			return &models.Board{Id: got, Owner: "owner"}, nil
		},
		DeleteBoardFn: func(b uuid.UUID) error {
			if b != id {
				t.Errorf("board = %v, want %v", b, id)
			}
			called = true
			return nil
		},
	}

	if err := New(db, &purgeRecorder{}).DeleteBoard(id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("DeleteBoard was not delegated")
	}
}

func TestCollaborators_Delegate(t *testing.T) {
	id := uuid.New()
	var added, removed string
	db := &mocks.MockDB{
		AddCollaboratorToBoardFn: func(_ uuid.UUID, cognitoID string) error {
			added = cognitoID
			return nil
		},
		RemoveCollaboratorFromBoardFn: func(_ uuid.UUID, cognitoID string) error {
			removed = cognitoID
			return nil
		},
	}

	svc := New(db, &purgeRecorder{})
	if err := svc.AddCollaboratorToBoard(id, "c1"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := svc.RemoveCollaboratorFromBoard(id, "c2"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if added != "c1" || removed != "c2" {
		t.Errorf("added=%q removed=%q, want c1/c2", added, removed)
	}
}

func TestUpdateBoardName_Delegates(t *testing.T) {
	var gotName string
	db := &mocks.MockDB{
		UpdateBoardNameFn: func(_ uuid.UUID, name string) error {
			gotName = name
			return nil
		},
	}

	if err := New(db, &purgeRecorder{}).UpdateBoardName(uuid.New(), "Renamed"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotName != "Renamed" {
		t.Errorf("name = %q, want Renamed", gotName)
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

	if err := New(db, purger).DeleteBoard(id); err != nil {
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
	db := &mocks.MockDB{
		RemoveCollaboratorFromBoardFn: func(got uuid.UUID, user string) error {
			removed = got == id && user == "ana"
			return nil
		},
	}
	purger := &purgeRecorder{}

	if err := New(db, purger).RemoveCollaboratorFromBoard(id, "ana"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if !removed {
		t.Error("the collaborator was not removed")
	}
	if len(purger.purged) != 1 || purger.purged[0] != models.MemberScope(id, "ana") {
		t.Errorf("purged %v, want the member scope", purger.purged)
	}
}
