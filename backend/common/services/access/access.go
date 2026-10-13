package access

import (
	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

// Resolver holds the access rules. A board or a group gives roles itself; one
// that belongs to an organization also gives its admins the owner's role and
// its members a read-only one. Whichever is higher wins.
type Resolver struct {
	orgs infrastructure.OrgReader
}

var _ infrastructure.Access = Resolver{}

// New takes where organizations are read from; with nil, nobody belongs to
// any and only explicit roles count.
func New(orgs infrastructure.OrgReader) Resolver {
	return Resolver{orgs}
}

func (r Resolver) BoardRole(principal models.Principal, board *models.Board) models.BoardRole {
	if board == nil || principal.Anonymous() {
		return ""
	}

	explicit := board.ExplicitRole(principal.ID)
	if board.Org == nil || explicit == models.BoardOwner {
		return explicit
	}

	var inherited models.BoardRole
	switch r.OrgRole(principal, *board.Org) {
	case models.OrgRoleAdmin:
		inherited = models.BoardOwner
	case models.OrgRoleMember:
		inherited = models.BoardViewer
	}

	if inherited.AtLeast(explicit) {
		return inherited
	}
	return explicit
}

func (r Resolver) GroupRole(principal models.Principal, group *models.Group) models.GroupRole {
	if group == nil || principal.Anonymous() {
		return ""
	}

	if group.Owner == principal.ID {
		return models.GroupOwner
	}
	var explicit models.GroupRole
	if group.IsMember(principal.ID) {
		explicit = models.GroupMember
	}
	if group.Org == nil {
		return explicit
	}

	switch r.OrgRole(principal, *group.Org) {
	case models.OrgRoleAdmin:
		return models.GroupOwner
	case models.OrgRoleMember:
		if explicit == "" {
			return models.GroupViewer
		}
	}
	return explicit
}

func (r Resolver) OrgRole(principal models.Principal, id uuid.UUID) models.OrgRole {
	if r.orgs == nil || principal.Anonymous() {
		return ""
	}

	org, err := r.orgs.FindOrg(id)
	if err != nil {
		return ""
	}
	return org.RoleOf(principal.ID)
}

func (r Resolver) Orgs(principal models.Principal) ([]uuid.UUID, error) {
	if r.orgs == nil || principal.Anonymous() {
		return nil, nil
	}

	orgs, err := r.orgs.FindUserOrgs(principal.ID)
	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, 0, len(orgs))
	for _, org := range orgs {
		ids = append(ids, org.Id)
	}
	return ids, nil
}
