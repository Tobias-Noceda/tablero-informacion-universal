package orgs

import (
	"errors"
	"strings"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

var (
	ErrForbidden   = infrastructure.ErrForbidden
	ErrLastAdmin   = infrastructure.ErrLastOrgAdmin
	ErrInvalidName = errors.New("Organization name cannot be empty")
	ErrInvalidRole = errors.New("A member is an admin or a member")
	ErrOrgNotEmpty = errors.New("The organization still has boards or groups")
)

type OrgService struct {
	store infrastructure.OrgStore
	users infrastructure.UserReader
}

func New(store infrastructure.OrgStore, users infrastructure.UserReader) *OrgService {
	return &OrgService{store, users}
}

// member loads an organization the principal belongs to, with their role
// filled in. One that cannot be found is indistinguishable from one the
// principal is not in.
func (srv *OrgService) member(principal models.Principal, id uuid.UUID) (*models.Org, error) {
	if principal.Anonymous() {
		return nil, ErrForbidden
	}

	org, err := srv.store.FindOrg(id)
	if err != nil {
		return nil, ErrForbidden
	}

	org.Role = org.RoleOf(principal.ID)
	if org.Role == "" {
		return nil, ErrForbidden
	}
	return org, nil
}

func (srv *OrgService) admin(principal models.Principal, id uuid.UUID) (*models.Org, error) {
	org, err := srv.member(principal, id)
	if err != nil {
		return nil, err
	}
	if org.Role != models.OrgRoleAdmin {
		return nil, ErrForbidden
	}
	return org, nil
}

func (srv *OrgService) Create(principal models.Principal, name string) (*models.Org, error) {
	if principal.Anonymous() {
		return nil, ErrForbidden
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidName
	}

	org := &models.Org{
		Id:        uuid.New(),
		Name:      name,
		Members:   []models.OrgMember{{User: principal.ID, Role: models.OrgRoleAdmin}},
		CreatedAt: time.Now().UTC(),
	}
	if err := srv.store.CreateOrg(org); err != nil {
		return nil, err
	}

	org.Role = models.OrgRoleAdmin
	return org, nil
}

func (srv *OrgService) ListMine(principal models.Principal) ([]models.Org, error) {
	if principal.Anonymous() {
		return nil, ErrForbidden
	}

	orgs, err := srv.store.FindUserOrgs(principal.ID)
	if err != nil {
		return nil, err
	}
	if orgs == nil {
		orgs = []models.Org{}
	}
	for i := range orgs {
		orgs[i].Role = orgs[i].RoleOf(principal.ID)
	}
	return orgs, nil
}

// Get is the organization with everyone in it, in the order they joined.
func (srv *OrgService) Get(principal models.Principal, id uuid.UUID) (*models.OrgDetail, error) {
	org, err := srv.member(principal, id)
	if err != nil {
		return nil, err
	}

	found, err := srv.users.FindUsers(org.UserIDs())
	if err != nil {
		return nil, err
	}
	byID := make(map[string]models.UserSummary, len(found))
	for _, user := range found {
		byID[user.Id.String()] = user.Summary()
	}

	members := make([]models.OrgMemberSummary, 0, len(org.Members))
	for _, member := range org.Members {
		summary, ok := byID[member.User]
		if !ok {
			// An account that no longer exists still shows up, by id, so an
			// admin can remove it.
			summary.Id, _ = uuid.Parse(member.User)
		}
		members = append(members, models.OrgMemberSummary{User: summary, Role: member.Role})
	}

	return &models.OrgDetail{Id: org.Id, Name: org.Name, Role: org.Role, Members: members, CreatedAt: org.CreatedAt}, nil
}

func (srv *OrgService) Rename(principal models.Principal, id uuid.UUID, name string) error {
	if _, err := srv.admin(principal, id); err != nil {
		return err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidName
	}
	return srv.store.RenameOrg(id, name)
}

// Delete removes an organization that no longer owns anything: its boards and
// groups have to be deleted first, so nothing is left reachable by nobody.
func (srv *OrgService) Delete(principal models.Principal, id uuid.UUID) error {
	if _, err := srv.admin(principal, id); err != nil {
		return err
	}

	for _, count := range []func(uuid.UUID) (int64, error){srv.store.CountOrgBoards, srv.store.CountOrgGroups} {
		n, err := count(id)
		if err != nil {
			return err
		}
		if n > 0 {
			return ErrOrgNotEmpty
		}
	}

	return srv.store.DeleteOrg(id)
}

// SetMember adds the account registered under email, or changes its role.
// The store refuses to demote the last admin.
func (srv *OrgService) SetMember(principal models.Principal, id uuid.UUID, email string, role models.OrgRole) (*models.OrgMemberSummary, error) {
	if _, err := srv.admin(principal, id); err != nil {
		return nil, err
	}
	if !role.Valid() {
		return nil, ErrInvalidRole
	}

	user, err := srv.users.FindUserByEmail(models.NormalizeEmail(email))
	if err != nil {
		return nil, err
	}

	if err := srv.store.SetOrgMember(id, user.Id.String(), role); err != nil {
		return nil, err
	}
	return &models.OrgMemberSummary{User: user.Summary(), Role: role}, nil
}

// RemoveMember is an admin's to decide, except that anyone may leave. The
// last admin can do neither.
func (srv *OrgService) RemoveMember(principal models.Principal, id uuid.UUID, user string) error {
	org, err := srv.member(principal, id)
	if err != nil {
		return err
	}
	if org.Role != models.OrgRoleAdmin && user != principal.ID {
		return ErrForbidden
	}

	return srv.store.RemoveOrgMember(id, user)
}
