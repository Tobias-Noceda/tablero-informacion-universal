package groups

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
	ErrOrgNotFound = infrastructure.ErrOrgNotFound
	ErrInvalidName = errors.New("Group name cannot be empty")
	ErrInvalidUser = errors.New("Member id cannot be empty")
)

type GroupService struct {
	store   infrastructure.GroupStore
	secrets infrastructure.ScopePurger
	access  infrastructure.Access
	plans   infrastructure.Entitlements
}

func New(store infrastructure.GroupStore, secrets infrastructure.ScopePurger, access infrastructure.Access, plans infrastructure.Entitlements) *GroupService {
	return &GroupService{store, secrets, access, plans}
}

// Create makes the caller the group's owner. A group created in an
// organization the caller belongs to is also owned by its admins. Either
// way it counts against the caller's plan, like the secrets it will hold.
func (srv *GroupService) Create(principal models.Principal, name string, org *uuid.UUID) (*models.Group, error) {
	if principal.Anonymous() {
		return nil, ErrForbidden
	}
	if org != nil && srv.access.OrgRole(principal, *org) == "" {
		return nil, ErrOrgNotFound
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidName
	}

	plan, err := srv.plans.ForUser(principal.ID)
	if err != nil {
		return nil, err
	}
	owned, err := srv.store.CountOwnedGroups(principal.ID)
	if err != nil {
		return nil, err
	}
	if err := plan.Admits(models.LimitGroups, int(owned), plan.Limits.Groups); err != nil {
		return nil, err
	}

	group := &models.Group{
		Id:        uuid.New(),
		Name:      name,
		Owner:     principal.ID,
		Members:   []string{},
		Org:       org,
		CreatedAt: time.Now().UTC(),
	}

	if err := srv.store.CreateGroup(group); err != nil {
		return nil, err
	}

	group.Role = models.GroupOwner
	return group, nil
}

func (srv *GroupService) Get(principal models.Principal, id uuid.UUID) (*models.Group, error) {
	return srv.require(principal, id, models.GroupViewer)
}

func (srv *GroupService) ListMine(principal models.Principal) ([]models.Group, error) {
	if principal.Anonymous() {
		return nil, ErrForbidden
	}

	orgs, err := srv.access.Orgs(principal)
	if err != nil {
		return nil, err
	}
	groups, err := srv.store.FindUserGroups(principal.ID, orgs)
	if err != nil {
		return nil, err
	}
	if groups == nil {
		groups = []models.Group{}
	}
	for i := range groups {
		groups[i].Role = srv.access.GroupRole(principal, &groups[i])
	}

	return groups, nil
}

// AddMember lists userID in the group. Someone new takes one of the seats
// the owner's plan gives it, the owner's own included.
func (srv *GroupService) AddMember(principal models.Principal, id uuid.UUID, userID string) error {
	group, err := srv.require(principal, id, models.GroupOwner)
	if err != nil {
		return err
	}
	if userID == "" {
		return ErrInvalidUser
	}

	if !group.IsMember(userID) {
		plan, err := srv.plans.ForUser(group.Owner)
		if err != nil {
			return err
		}
		if err := plan.Admits(models.LimitSeats, len(group.Members)+1, plan.Limits.Seats); err != nil {
			return err
		}
	}

	return srv.store.AddGroupMember(id, userID)
}

func (srv *GroupService) RemoveMember(principal models.Principal, id uuid.UUID, userID string) error {
	if _, err := srv.require(principal, id, models.GroupOwner); err != nil {
		return err
	}

	return srv.store.RemoveGroupMember(id, userID)
}

// Delete removes the group and everything it owned in the vault.
func (srv *GroupService) Delete(principal models.Principal, id uuid.UUID) error {
	if _, err := srv.require(principal, id, models.GroupOwner); err != nil {
		return err
	}

	if err := srv.store.DeleteGroup(id); err != nil {
		return err
	}

	return srv.secrets.Purge(models.GroupScope(id))
}

// require loads a group in which the principal holds at least min, with the
// principal's role filled in. One that cannot be found is indistinguishable
// from one the principal may not see.
func (srv *GroupService) require(principal models.Principal, id uuid.UUID, min models.GroupRole) (*models.Group, error) {
	group, err := srv.store.FindGroup(id)
	if err != nil {
		return nil, ErrForbidden
	}

	group.Role = srv.access.GroupRole(principal, group)
	if !group.Role.AtLeast(min) {
		return nil, ErrForbidden
	}

	return group, nil
}
