package infrastructure

import "github.com/Secreto31126/tesis/common/models"

// Access answers what a principal may do on a board or in a group. Every
// policy asks it rather than reading the members itself, so a rule that
// grants a role from elsewhere (an organization) is added in one place.
type Access interface {
	// BoardRole is the principal's role on board, or "" for none.
	BoardRole(principal models.Principal, board *models.Board) models.BoardRole
	// GroupRole is the principal's role in group, or "" for none.
	GroupRole(principal models.Principal, group *models.Group) models.GroupRole
}
