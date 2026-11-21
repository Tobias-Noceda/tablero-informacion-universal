package boards

import (
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/services/access"
	"github.com/google/uuid"
)

// CONC-04: two requests creating the last board a plan allows take turns,
// so the second counts the first's board.
func TestCreateBoard_CountsAndCreatesInItsTurn(t *testing.T) {
	var s mocks.Steps
	db := &mocks.MockDB{
		CountPersonalBoardsFn: func(string) (int64, error) { s.Add("count"); return 0, nil },
		CreateBoardFn: func(name, owner string, org *uuid.UUID) (*models.Board, error) {
			s.Add("create")
			return &models.Board{Id: uuid.New(), Name: name, Owner: owner}, nil
		},
	}
	svc := New(db, &purgeRecorder{}, access.New(nil), users(), &mocks.CountingLimiter{}, nil, s.Plans())

	if _, err := svc.CreateBoard(ana, "mine", nil); err != nil {
		t.Fatal(err)
	}

	if !s.In("hold "+models.LimitBoards+" user:"+ana.ID, "count", "create", "release") {
		t.Errorf("steps = %v", s.List())
	}
}

// The board read before the turn may already be stale: whoever adds someone
// counts the people on the board as it is once the turn is theirs.
func TestSetMember_CountsSeatsOnTheBoardAsItIsInItsTurn(t *testing.T) {
	var s mocks.Steps
	id := uuid.New()
	db := boardDB(id)
	find := db.FindBoardFn
	db.FindBoardFn = func(got uuid.UUID) (*models.Board, error) {
		s.Add("find")
		return find(got)
	}
	db.SetBoardMemberFn = func(uuid.UUID, string, models.BoardRole) error { s.Add("set"); return nil }
	svc := New(db, &purgeRecorder{}, access.New(nil), users(), &mocks.CountingLimiter{}, nil, s.Plans())

	if _, err := svc.SetMember(owner, id, eveUser.Email, models.BoardEditor); err != nil {
		t.Fatal(err)
	}

	if !s.In("hold "+models.LimitSeats+" board:"+id.String(), "find", "set", "release") {
		t.Errorf("steps = %v", s.List())
	}
}
