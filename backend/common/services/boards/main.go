package boards

import (
	"errors"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

var (
	ErrForbidden   = infrastructure.ErrForbidden
	ErrInvalidRole = errors.New("A member is an editor or a viewer")
	ErrOwnerRole   = errors.New("The owner's role cannot change")
)

type BoardService struct {
	db      infrastructure.Database
	secrets infrastructure.ScopePurger
	access  infrastructure.Access
	users   infrastructure.UserReader
}

func New(db infrastructure.Database, secrets infrastructure.ScopePurger, access infrastructure.Access, users infrastructure.UserReader) *BoardService {
	return &BoardService{db, secrets, access, users}
}

// require loads a board on which the principal holds at least min, with the
// principal's role filled in. A board that cannot be found is
// indistinguishable from one the principal may not see.
func (srv *BoardService) require(principal models.Principal, id uuid.UUID, min models.BoardRole) (*models.Board, error) {
	board, err := srv.db.FindBoard(id)
	if err != nil || board == nil {
		return nil, ErrForbidden
	}

	board.Role = srv.access.BoardRole(principal, board)
	if !board.Role.AtLeast(min) {
		return nil, ErrForbidden
	}
	return board, nil
}

func (srv *BoardService) GetUserBoards(principal models.Principal) ([]models.Board, error) {
	if principal.Anonymous() {
		return nil, ErrForbidden
	}

	boards, err := srv.db.FindUserBoards(principal.ID)
	if err != nil {
		return nil, err
	}
	for i := range boards {
		boards[i].Role = srv.access.BoardRole(principal, &boards[i])
	}
	return boards, nil
}

func (srv *BoardService) GetBoard(principal models.Principal, id uuid.UUID) (*models.Board, error) {
	return srv.require(principal, id, models.BoardViewer)
}

func (srv *BoardService) CreateBoard(principal models.Principal, name string) (*models.Board, error) {
	if principal.Anonymous() {
		return nil, ErrForbidden
	}

	board, err := srv.db.CreateBoard(name, principal.ID)
	if err != nil {
		return nil, err
	}
	board.Role = models.BoardOwner
	return board, nil
}

func (srv *BoardService) DeleteBoard(principal models.Principal, id uuid.UUID) error {
	board, err := srv.require(principal, id, models.BoardOwner)
	if err != nil {
		return err
	}

	if err := srv.db.DeleteBoard(id); err != nil {
		return err
	}

	if err := srv.secrets.Purge(models.BoardScope(id)); err != nil {
		return err
	}

	for _, user := range board.Everyone() {
		if err := srv.secrets.Purge(models.MemberScope(id, user)); err != nil {
			return err
		}
	}

	return nil
}

func (srv *BoardService) GetBoardPostIts(principal models.Principal, id uuid.UUID) ([]models.PostIts, error) {
	if _, err := srv.require(principal, id, models.BoardViewer); err != nil {
		return nil, err
	}
	return srv.db.FindBoardPostIts(id)
}

// Members lists everyone on the board, the owner first.
func (srv *BoardService) Members(principal models.Principal, id uuid.UUID) ([]models.BoardMemberSummary, error) {
	board, err := srv.require(principal, id, models.BoardViewer)
	if err != nil {
		return nil, err
	}

	ids := board.Everyone()
	found, err := srv.users.FindUsers(ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]models.UserSummary, len(found))
	for _, user := range found {
		byID[user.Id.String()] = user.Summary()
	}

	members := make([]models.BoardMemberSummary, 0, len(ids))
	for _, user := range ids {
		summary, ok := byID[user]
		if !ok {
			// An account that no longer exists still shows up, by id, so the
			// owner can remove it.
			summary.Id, _ = uuid.Parse(user)
		}
		members = append(members, models.BoardMemberSummary{User: summary, Role: board.ExplicitRole(user)})
	}
	return members, nil
}

// SetMember adds the account registered under email, or changes its role.
func (srv *BoardService) SetMember(principal models.Principal, id uuid.UUID, email string, role models.BoardRole) (*models.BoardMemberSummary, error) {
	board, err := srv.require(principal, id, models.BoardOwner)
	if err != nil {
		return nil, err
	}
	if !role.Assignable() {
		return nil, ErrInvalidRole
	}

	user, err := srv.users.FindUserByEmail(models.NormalizeEmail(email))
	if err != nil {
		return nil, err
	}
	if user.Id.String() == board.Owner {
		return nil, ErrOwnerRole
	}

	if err := srv.db.SetBoardMember(id, user.Id.String(), role); err != nil {
		return nil, err
	}
	return &models.BoardMemberSummary{User: user.Summary(), Role: role}, nil
}

// RemoveMember is the owner's to decide, except that anyone may leave a board
// on their own. What the member kept on the board goes with them.
func (srv *BoardService) RemoveMember(principal models.Principal, id uuid.UUID, user string) error {
	board, err := srv.require(principal, id, models.BoardViewer)
	if err != nil {
		return err
	}
	if board.Role != models.BoardOwner && user != principal.ID {
		return ErrForbidden
	}
	if user == board.Owner {
		return ErrOwnerRole
	}

	if err := srv.db.RemoveBoardMember(id, user); err != nil {
		return err
	}

	return srv.secrets.Purge(models.MemberScope(id, user))
}

func (srv *BoardService) UpdateBoardName(principal models.Principal, id uuid.UUID, name string) error {
	if _, err := srv.require(principal, id, models.BoardOwner); err != nil {
		return err
	}
	return srv.db.UpdateBoardName(id, name)
}

func (srv *BoardService) ConnectPostIts(principal models.Principal, boardID, source, target uuid.UUID) (*models.Strand, error) {
	if _, err := srv.require(principal, boardID, models.BoardEditor); err != nil {
		return nil, err
	}
	return srv.db.ConnectPostIts(boardID, source, target)
}

func (srv *BoardService) DisconnectPostIts(principal models.Principal, boardID, strandID uuid.UUID) error {
	if _, err := srv.require(principal, boardID, models.BoardEditor); err != nil {
		return err
	}
	return srv.db.DisconnectPostIts(boardID, strandID)
}
