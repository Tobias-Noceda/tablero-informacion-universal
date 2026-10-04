//go:build integration

package mongo

import (
	"testing"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

func TestMongo_BoardMembers(t *testing.T) {
	db := integrationDB(t)

	board, err := db.CreateBoard("it", "owner")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := db.SetBoardMember(board.Id, "ana", models.BoardViewer); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := db.SetBoardMember(board.Id, "ana", models.BoardEditor); err != nil {
		t.Fatalf("promote: %v", err)
	}
	if err := db.SetBoardMember(board.Id, "bob", models.BoardViewer); err != nil {
		t.Fatalf("add bob: %v", err)
	}

	found, err := db.FindBoard(board.Id)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	want := []models.BoardMember{{User: "ana", Role: models.BoardEditor}, {User: "bob", Role: models.BoardViewer}}
	if len(found.Members) != 2 || found.Members[0] != want[0] || found.Members[1] != want[1] {
		t.Errorf("members = %+v, want %+v (changing a role must not add a second entry)", found.Members, want)
	}

	for _, user := range []string{"owner", "ana", "bob"} {
		boards, err := db.FindUserBoards(user)
		if err != nil {
			t.Fatalf("boards of %s: %v", user, err)
		}
		if len(boards) != 1 || boards[0].Id != board.Id {
			t.Errorf("%s's boards = %+v, want the board", user, boards)
		}
	}
	if boards, _ := db.FindUserBoards("eve"); len(boards) != 0 {
		t.Errorf("eve's boards = %+v, want none", boards)
	}

	if err := db.RemoveBoardMember(board.Id, "ana"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if boards, _ := db.FindUserBoards("ana"); len(boards) != 0 {
		t.Errorf("ana still lists %+v", boards)
	}

	if err := db.SetBoardMember(uuid.New(), "ana", models.BoardViewer); err == nil {
		t.Error("adding a member to a board that does not exist succeeded")
	}
}

func TestMongo_BoardIndexes(t *testing.T) {
	db := integrationDB(t)

	ctx, cancel := timeout()
	defer cancel()

	cursor, err := db.boards.Indexes().List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var indexes []struct {
		Name string `bson:"name"`
	}
	if err := cursor.All(ctx, &indexes); err != nil {
		t.Fatalf("decode: %v", err)
	}

	names := map[string]bool{}
	for _, index := range indexes {
		names[index.Name] = true
	}
	for _, want := range []string{"owner_1", "members.user_1"} {
		if !names[want] {
			t.Errorf("index %s missing, have %v", want, names)
		}
	}
}
