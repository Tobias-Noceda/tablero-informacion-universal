package postits

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

func TestDuplicatePostIt_AnswersTheCopy(t *testing.T) {
	card, board := uuid.New(), uuid.New()
	db := &mocks.MockDB{
		FindBoardFn: func(id uuid.UUID) (*models.Board, error) {
			return &models.Board{Id: id, Owner: owner, PostIts: []models.BoardPostIt{
				{Id: card, Title: models.BoardPostItTitle{Text: "Note"}, Position: models.Position{X: 10, Y: 20}},
			}}, nil
		},
		FindPostItFn: func(id uuid.UUID) (*models.PostIts, error) {
			return &models.PostIts{Id: id, Board: board, RunAs: owner}, nil
		},
		CreatePostItFn: func(p *models.PostIts, _ models.BoardPostItTitle, _ string, _ models.Position) (*models.PostIts, error) {
			p.Id = uuid.New()
			return p, nil
		},
	}
	r := setupRouter(db, nil, nil)

	w := do(r, http.MethodPost, "/post-its/"+card.String()+"/duplicate", "")
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", w.Code, w.Body.String())
	}
	var got models.BoardPostIt
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Id == card || got.Id == uuid.Nil || got.Title.Text != "Note" || got.Position != (models.Position{X: 50, Y: 60}) {
		t.Errorf("answered %+v", got)
	}
}

func TestDuplicatePostIt_RefusesBadIdsAndStrangers(t *testing.T) {
	r := setupRouter(nil, nil, nil)
	if w := do(r, http.MethodPost, "/post-its/nope/duplicate", ""); w.Code != http.StatusBadRequest {
		t.Errorf("bad id: status = %d, want 400", w.Code)
	}
	if w := doAs(r, "stranger", http.MethodPost, "/post-its/"+uuid.NewString()+"/duplicate", ""); w.Code != http.StatusNotFound {
		t.Errorf("stranger: status = %d, want 404", w.Code)
	}
}
