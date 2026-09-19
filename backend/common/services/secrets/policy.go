package secrets

import (
	"errors"
	"slices"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

var ErrForbidden = errors.New("Not allowed to manage these secrets")

type policy struct {
	boards infrastructure.BoardReader
}

var _ infrastructure.ScopePolicy = (*policy)(nil)

func NewPolicy(boards infrastructure.BoardReader) *policy {
	return &policy{boards}
}

func (p *policy) CanManage(principal models.Principal, scope models.SecretScope) error {
	return p.check(principal, scope, true)
}

func (p *policy) CanView(principal models.Principal, scope models.SecretScope) error {
	return p.check(principal, scope, false)
}

func (p *policy) check(principal models.Principal, scope models.SecretScope, manage bool) error {
	if !scope.Valid() {
		return ErrForbidden
	}

	switch scope.Kind {
	case models.ScopeBoard:
		return p.checkBoard(principal, scope, manage)
	case models.ScopeUser:
		if principal.Anonymous() || principal.ID != scope.Owner {
			return ErrForbidden
		}
		return nil
	case models.ScopeSystem:
		// Nobody can be told apart until authentication exists. This is where
		// the admin role check goes once it does.
		return nil
	default:
		return ErrForbidden
	}
}

func (p *policy) checkBoard(principal models.Principal, scope models.SecretScope, manage bool) error {
	if principal.Anonymous() {
		return ErrForbidden
	}

	board, err := p.boards.FindBoard(uuid.MustParse(scope.Owner))
	if err != nil {
		return ErrForbidden
	}

	if board.Owner == principal.ID {
		return nil
	}

	if !manage && slices.Contains(board.Collaborators, principal.ID) {
		return nil
	}

	return ErrForbidden
}
