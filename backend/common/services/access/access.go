package access

import (
	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
)

// Resolver holds the access rules. Today a board's role comes from the board
// alone; organization roles join here in phase 17.
type Resolver struct{}

var _ infrastructure.Access = Resolver{}

func New() Resolver {
	return Resolver{}
}

func (Resolver) BoardRole(principal models.Principal, board *models.Board) models.BoardRole {
	if board == nil || principal.Anonymous() {
		return ""
	}
	return board.ExplicitRole(principal.ID)
}

func (Resolver) GroupRole(principal models.Principal, group *models.Group) models.GroupRole {
	if group == nil || principal.Anonymous() {
		return ""
	}
	if group.Owner == principal.ID {
		return models.GroupOwner
	}
	if group.IsMember(principal.ID) {
		return models.GroupMember
	}
	return ""
}
