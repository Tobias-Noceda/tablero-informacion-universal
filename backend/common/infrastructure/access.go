package infrastructure

import (
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

// Access answers what a principal may do on a board, in a group or in an
// organization. Every policy asks it rather than reading the members itself,
// so a rule that grants a role from elsewhere is added in one place.
type Access interface {
	// BoardRole is the principal's role on board, or "" for none.
	BoardRole(principal models.Principal, board *models.Board) models.BoardRole
	// GroupRole is the principal's role in group, or "" for none.
	GroupRole(principal models.Principal, group *models.Group) models.GroupRole
	// OrgRole is the principal's role in the organization, or "" for none
	// (also when it does not exist).
	OrgRole(principal models.Principal, org uuid.UUID) models.OrgRole
	// Orgs are the organizations the principal belongs to, whose boards and
	// groups reach them without being shared.
	Orgs(principal models.Principal) ([]uuid.UUID, error)
}
