package boards

import (
	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

type BoardService struct {
	db infrastructure.Database
}

func New(db infrastructure.Database) *BoardService {
	return &BoardService{db}
}

func (srv *BoardService) GetUserBoards(cognito_id string) ([]models.Board, error) {
	return srv.db.FindUserBoards(cognito_id)
}

func (srv *BoardService) GetBoard(id uuid.UUID) (*models.Board, error) {
	return srv.db.FindBoard(id)
}

func (srv *BoardService) CreateBoard(name, owner string) (*models.Board, error) {
	return srv.db.CreateBoard(name, owner)
}

func (srv *BoardService) DeleteBoard(id uuid.UUID) error {
	return srv.db.DeleteBoard(id)
}

func (srv *BoardService) GetBoardPostIts(id uuid.UUID) ([]models.PostIts, error) {
	return srv.db.FindBoardPostIts(id)
}

func (srv *BoardService) AddCollaboratorToBoard(boardID uuid.UUID, cognitoID string) error {
	return srv.db.AddCollaboratorToBoard(boardID, cognitoID)
}

func (srv *BoardService) RemoveCollaboratorFromBoard(boardID uuid.UUID, cognitoID string) error {
	return srv.db.RemoveCollaboratorFromBoard(boardID, cognitoID)
}

func (srv *BoardService) UpdateBoardName(id uuid.UUID, name string) error {
	return srv.db.UpdateBoardName(id, name)
}

func (srv *BoardService) ConnectPostIts(boardID, source, target uuid.UUID) error {
	return srv.db.ConnectPostIts(boardID, source, target)
}

func (srv *BoardService) DisconnectPostIts(boardID, source, target uuid.UUID) error {
	return srv.db.DisconnectPostIts(boardID, source, target)
}
