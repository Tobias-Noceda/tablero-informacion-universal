package boards

import (
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

	svc := New(db)
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
		ConnectPostItsFn: func(b, s, t uuid.UUID) error {
			gb, gs, gt = b, s, t
			return nil
		},
	}

	svc := New(db)
	if err := svc.ConnectPostIts(board, src, tgt); err != nil {
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

	svc := New(db)
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

	svc := New(db)
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

	board, err := New(db).GetBoard(id)
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

	got, err := New(db).GetBoardPostIts(id)
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
		DeleteBoardFn: func(b uuid.UUID) error {
			if b != id {
				t.Errorf("board = %v, want %v", b, id)
			}
			called = true
			return nil
		},
	}

	if err := New(db).DeleteBoard(id); err != nil {
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

	svc := New(db)
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

	if err := New(db).UpdateBoardName(uuid.New(), "Renamed"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotName != "Renamed" {
		t.Errorf("name = %q, want Renamed", gotName)
	}
}
