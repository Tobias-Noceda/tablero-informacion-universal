package secrets

import (
	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

var ErrForbidden = infrastructure.ErrForbidden

type policy struct {
	boards infrastructure.BoardReader
	groups infrastructure.GroupReader
}

var _ infrastructure.ScopePolicy = (*policy)(nil)

func NewPolicy(boards infrastructure.BoardReader, groups infrastructure.GroupReader) *policy {
	return &policy{boards, groups}
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
	case models.ScopeMember:
		return p.checkMember(principal, scope)
	case models.ScopeGroup:
		return p.checkGroup(principal, scope, manage)
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
	board, err := p.board(principal, uuid.MustParse(scope.Owner))
	if err != nil {
		return err
	}

	if board.Owner == principal.ID {
		return nil
	}

	if !manage && board.IsMember(principal.ID) {
		return nil
	}

	return ErrForbidden
}

func (p *policy) checkMember(principal models.Principal, scope models.SecretScope) error {
	boardID, userID, _ := scope.Member()
	if principal.ID != userID {
		return ErrForbidden
	}

	_, err := p.board(principal, boardID)
	return err
}

func (p *policy) checkGroup(principal models.Principal, scope models.SecretScope, manage bool) error {
	group, err := p.group(principal, uuid.MustParse(scope.Owner))
	if err != nil {
		return err
	}

	if manage && group.Owner != principal.ID {
		return ErrForbidden
	}

	return nil
}

// CanUse decides whether principal may bind secret into a card on board:
// either the scope itself admits them, or a grant does. It is evaluated
// again on every execution, so losing access stops the card.
func (p *policy) CanUse(principal models.Principal, boardID uuid.UUID, secret *models.Secret) error {
	if !secret.Scope.Valid() || principal.Anonymous() || secret.Scope.Kind == models.ScopeSystem {
		return ErrForbidden
	}

	if p.owns(principal, boardID, secret.Scope) == nil {
		return nil
	}

	for _, grant := range secret.Grants {
		if grant.AppliesTo(boardID) && p.reaches(principal, boardID, grant.To) == nil {
			return nil
		}
	}

	return ErrForbidden
}

func (p *policy) owns(principal models.Principal, boardID uuid.UUID, scope models.SecretScope) error {
	switch scope.Kind {
	case models.ScopeBoard:
		if scope.Owner != boardID.String() {
			return ErrForbidden
		}
		_, err := p.board(principal, boardID)
		return err
	case models.ScopeMember:
		memberBoard, userID, _ := scope.Member()
		if memberBoard != boardID || principal.ID != userID {
			return ErrForbidden
		}
		_, err := p.board(principal, boardID)
		return err
	case models.ScopeUser:
		if principal.ID != scope.Owner {
			return ErrForbidden
		}
		return nil
	case models.ScopeGroup:
		_, err := p.group(principal, uuid.MustParse(scope.Owner))
		return err
	default:
		return ErrForbidden
	}
}

func (p *policy) reaches(principal models.Principal, boardID uuid.UUID, audience models.Audience) error {
	switch audience.Kind {
	case models.AudienceUser:
		if principal.ID != audience.ID {
			return ErrForbidden
		}
		return nil
	case models.AudienceGroup:
		id, err := uuid.Parse(audience.ID)
		if err != nil {
			return ErrForbidden
		}
		_, err = p.group(principal, id)
		return err
	case models.AudienceBoard:
		if audience.ID != boardID.String() {
			return ErrForbidden
		}
		_, err := p.board(principal, boardID)
		return err
	default:
		return ErrForbidden
	}
}

// board loads a board the principal is a member of. A board that cannot be
// found is indistinguishable from one the principal may not touch.
func (p *policy) board(principal models.Principal, id uuid.UUID) (*models.Board, error) {
	if principal.Anonymous() {
		return nil, ErrForbidden
	}

	board, err := p.boards.FindBoard(id)
	if err != nil || !board.IsMember(principal.ID) {
		return nil, ErrForbidden
	}

	return board, nil
}

// group loads a group the principal belongs to, with the same rule as board.
func (p *policy) group(principal models.Principal, id uuid.UUID) (*models.Group, error) {
	if principal.Anonymous() {
		return nil, ErrForbidden
	}

	group, err := p.groups.FindGroup(id)
	if err != nil || !group.IsMember(principal.ID) {
		return nil, ErrForbidden
	}

	return group, nil
}
