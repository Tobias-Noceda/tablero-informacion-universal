package postits

import (
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/services/access"
	"github.com/google/uuid"
)

// CONC-04: a board's last card is counted on the board as it is in the
// board's turn, and created before the turn ends.
func TestCreatePostIt_CountsTheBoardsCardsInItsTurn(t *testing.T) {
	var s mocks.Steps
	board := uuid.New()
	db := &mocks.MockDB{
		FindBoardFn: func(id uuid.UUID) (*models.Board, error) {
			s.Add("find")
			return &models.Board{Id: id, Owner: boardOwner.ID}, nil
		},
		CreatePostItFn: func(postIt *models.PostIts, title models.BoardPostItTitle, ptype string, pos models.Position) (*models.PostIts, error) {
			s.Add("create")
			postIt.Id = uuid.New()
			return postIt, nil
		},
	}
	svc := New(db, &mocks.MockCache{}, &mocks.MockExecuter{}, &mocks.MockSecretResolver{}, access.New(nil), builtins(), nil, s.Plans(), &mocks.CountingLimiter{})

	_, err := svc.CreatePostIt(boardOwner, &models.BoardPostItTitle{Text: "note"}, &models.PostIts{Board: board, WellKnown: "static_card", Params: map[string]string{"text": "hi"}})
	if err != nil {
		t.Fatal(err)
	}

	if !s.In("find", "hold "+models.LimitNodes+" board:"+board.String(), "find", "create") {
		t.Errorf("steps = %v", s.List())
	}
}
