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
	access infrastructure.Access
}

var _ infrastructure.ScopePolicy = (*policy)(nil)

func NewPolicy(boards infrastructure.BoardReader, groups infrastructure.GroupReader, access infrastructure.Access) *policy {
	return &policy{boards, groups, access}
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
		// The board's credentials are for building cards: editors see their
		// names, only the owner changes them, a viewer does not learn them.
		min := models.BoardEditor
		if manage {
			min = models.BoardOwner
		}
		return p.onBoard(principal, uuid.MustParse(scope.Owner), min)
	case models.ScopeMember:
		// What someone keeps on a board is for cards, so it takes an editor.
		boardID, userID, _ := scope.Member()
		if principal.ID != userID {
			return ErrForbidden
		}
		return p.onBoard(principal, boardID, models.BoardEditor)
	case models.ScopeGroup:
		// A group's secrets are for the people it lists; someone who only
		// sees the group through its organization does not learn them.
		min := models.GroupMember
		if manage {
			min = models.GroupOwner
		}
		if !p.groupRole(principal, uuid.MustParse(scope.Owner)).AtLeast(min) {
			return ErrForbidden
		}
		return nil
	case models.ScopeUser:
		if principal.Anonymous() || principal.ID != scope.Owner {
			return ErrForbidden
		}
		return nil
	case models.ScopeSystem:
		if !principal.Admin {
			return ErrForbidden
		}
		return nil
	default:
		return ErrForbidden
	}
}

// CanUse decides whether principal may bind secret into a card on board:
// they must be able to edit the board, and then either the scope itself
// admits them or a grant does. It is evaluated again on every execution, so
// losing access, or being demoted to viewer, stops the card.
func (p *policy) CanUse(principal models.Principal, boardID uuid.UUID, secret *models.Secret) error {
	if !secret.Scope.Valid() || principal.Anonymous() || secret.Scope.Kind == models.ScopeSystem {
		return ErrForbidden
	}

	if err := p.onBoard(principal, boardID, models.BoardEditor); err != nil {
		return err
	}

	if p.owns(principal, boardID, secret.Scope) {
		return nil
	}

	for _, grant := range secret.Grants {
		if grant.AppliesTo(boardID) && p.reaches(principal, boardID, grant.To) {
			return nil
		}
	}

	return ErrForbidden
}

// owns tells whether the secret's own scope admits principal on boardID. The
// caller has already checked that principal edits boardID.
func (p *policy) owns(principal models.Principal, boardID uuid.UUID, scope models.SecretScope) bool {
	switch scope.Kind {
	case models.ScopeBoard:
		return scope.Owner == boardID.String()
	case models.ScopeMember:
		memberBoard, userID, _ := scope.Member()
		return memberBoard == boardID && principal.ID == userID
	case models.ScopeUser:
		return principal.ID == scope.Owner
	case models.ScopeGroup:
		return p.groupRole(principal, uuid.MustParse(scope.Owner)).AtLeast(models.GroupMember)
	default:
		return false
	}
}

// reaches tells whether a grant's audience includes principal on boardID. A
// grant to a board reaches whoever may bind on it, which CanUse already
// checked.
func (p *policy) reaches(principal models.Principal, boardID uuid.UUID, audience models.Audience) bool {
	switch audience.Kind {
	case models.AudienceUser:
		return principal.ID == audience.ID
	case models.AudienceGroup:
		id, err := uuid.Parse(audience.ID)
		return err == nil && p.groupRole(principal, id).AtLeast(models.GroupMember)
	case models.AudienceBoard:
		return audience.ID == boardID.String()
	default:
		return false
	}
}

// onBoard requires principal to hold at least min on the board. A board that
// cannot be found is indistinguishable from one the principal may not touch.
func (p *policy) onBoard(principal models.Principal, id uuid.UUID, min models.BoardRole) error {
	if principal.Anonymous() {
		return ErrForbidden
	}

	board, err := p.boards.FindBoard(id)
	if err != nil || !p.access.BoardRole(principal, board).AtLeast(min) {
		return ErrForbidden
	}

	return nil
}

func (p *policy) groupRole(principal models.Principal, id uuid.UUID) models.GroupRole {
	if principal.Anonymous() {
		return ""
	}

	group, err := p.groups.FindGroup(id)
	if err != nil {
		return ""
	}

	return p.access.GroupRole(principal, group)
}
