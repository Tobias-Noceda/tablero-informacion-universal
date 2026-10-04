package boards

import (
	"errors"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

var (
	ErrForbidden               = infrastructure.ErrForbidden
	ErrOwnerIsNotACollaborator = errors.New("The owner is not a collaborator")
)

type BoardService struct {
	db      infrastructure.Database
	secrets infrastructure.ScopePurger
}

func New(db infrastructure.Database, secrets infrastructure.ScopePurger) *BoardService {
	return &BoardService{db, secrets}
}

// member loads a board the principal belongs to. A board that cannot be found
// is indistinguishable from one the principal may not see.
func (srv *BoardService) member(principal models.Principal, id uuid.UUID) (*models.Board, error) {
	board, err := srv.db.FindBoard(id)
	if err != nil || board == nil || !board.IsMember(principal.ID) {
		return nil, ErrForbidden
	}
	return board, nil
}

func (srv *BoardService) owned(principal models.Principal, id uuid.UUID) (*models.Board, error) {
	board, err := srv.member(principal, id)
	if err != nil {
		return nil, err
	}
	if board.Owner != principal.ID {
		return nil, ErrForbidden
	}
	return board, nil
}

func (srv *BoardService) GetUserBoards(principal models.Principal) ([]models.Board, error) {
	if principal.Anonymous() {
		return nil, ErrForbidden
	}
	return srv.db.FindUserBoards(principal.ID)
}

func (srv *BoardService) GetBoard(principal models.Principal, id uuid.UUID) (*models.Board, error) {
	return srv.member(principal, id)
}

func (srv *BoardService) CreateBoard(principal models.Principal, name string) (*models.Board, error) {
	if principal.Anonymous() {
		return nil, ErrForbidden
	}
	return srv.db.CreateBoard(name, principal.ID)
}

func (srv *BoardService) DeleteBoard(principal models.Principal, id uuid.UUID) error {
	board, err := srv.owned(principal, id)
	if err != nil {
		return err
	}

	if err := srv.db.DeleteBoard(id); err != nil {
		return err
	}

	if err := srv.secrets.Purge(models.BoardScope(id)); err != nil {
		return err
	}

	for _, user := range append([]string{board.Owner}, board.Collaborators...) {
		if err := srv.secrets.Purge(models.MemberScope(id, user)); err != nil {
			return err
		}
	}

	return nil
}

func (srv *BoardService) GetBoardPostIts(principal models.Principal, id uuid.UUID) ([]models.PostIts, error) {
	if _, err := srv.member(principal, id); err != nil {
		return nil, err
	}
	return srv.db.FindBoardPostIts(id)
}

func (srv *BoardService) AddCollaboratorToBoard(principal models.Principal, boardID uuid.UUID, user string) error {
	if _, err := srv.owned(principal, boardID); err != nil {
		return err
	}
	return srv.db.AddCollaboratorToBoard(boardID, user)
}

// RemoveCollaboratorFromBoard is the owner's to decide, except that anyone
// may leave a board on their own.
func (srv *BoardService) RemoveCollaboratorFromBoard(principal models.Principal, boardID uuid.UUID, user string) error {
	board, err := srv.member(principal, boardID)
	if err != nil {
		return err
	}
	if board.Owner != principal.ID && user != principal.ID {
		return ErrForbidden
	}
	if user == board.Owner {
		return ErrOwnerIsNotACollaborator
	}

	if err := srv.db.RemoveCollaboratorFromBoard(boardID, user); err != nil {
		return err
	}

	return srv.secrets.Purge(models.MemberScope(boardID, user))
}

func (srv *BoardService) UpdateBoardName(principal models.Principal, id uuid.UUID, name string) error {
	if _, err := srv.owned(principal, id); err != nil {
		return err
	}
	return srv.db.UpdateBoardName(id, name)
}

func (srv *BoardService) ConnectPostIts(principal models.Principal, boardID, source, target uuid.UUID) (*models.Strand, error) {
	if _, err := srv.member(principal, boardID); err != nil {
		return nil, err
	}
	return srv.db.ConnectPostIts(boardID, source, target)
}

func (srv *BoardService) DisconnectPostIts(principal models.Principal, boardID, strandID uuid.UUID) error {
	if _, err := srv.member(principal, boardID); err != nil {
		return err
	}
	return srv.db.DisconnectPostIts(boardID, strandID)
}
