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
	ErrInvalidName = errors.New("Group name cannot be empty")
	ErrInvalidUser = errors.New("Member id cannot be empty")
)

type GroupService struct {
	store   infrastructure.GroupStore
	secrets infrastructure.ScopePurger
}

func New(store infrastructure.GroupStore, secrets infrastructure.ScopePurger) *GroupService {
	return &GroupService{store, secrets}
}

func (srv *GroupService) Create(principal models.Principal, name string) (*models.Group, error) {
	if principal.Anonymous() {
		return nil, ErrForbidden
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidName
	}

	group := &models.Group{
		Id:        uuid.New(),
		Name:      name,
		Owner:     principal.ID,
		Members:   []string{},
		CreatedAt: time.Now().UTC(),
	}

	if err := srv.store.CreateGroup(group); err != nil {
		return nil, err
	}

	return group, nil
}

func (srv *GroupService) Get(principal models.Principal, id uuid.UUID) (*models.Group, error) {
	return srv.member(principal, id)
}

func (srv *GroupService) ListMine(principal models.Principal) ([]models.Group, error) {
	if principal.Anonymous() {
		return nil, ErrForbidden
	}

	groups, err := srv.store.FindUserGroups(principal.ID)
	if err != nil {
		return nil, err
	}
	if groups == nil {
		groups = []models.Group{}
	}

	return groups, nil
}

func (srv *GroupService) AddMember(principal models.Principal, id uuid.UUID, userID string) error {
	if _, err := srv.owned(principal, id); err != nil {
		return err
	}
	if userID == "" {
		return ErrInvalidUser
	}

	return srv.store.AddGroupMember(id, userID)
}

func (srv *GroupService) RemoveMember(principal models.Principal, id uuid.UUID, userID string) error {
	if _, err := srv.owned(principal, id); err != nil {
		return err
	}

	return srv.store.RemoveGroupMember(id, userID)
}

// Delete removes the group and everything it owned in the vault.
func (srv *GroupService) Delete(principal models.Principal, id uuid.UUID) error {
	if _, err := srv.owned(principal, id); err != nil {
		return err
	}

	if err := srv.store.DeleteGroup(id); err != nil {
		return err
	}

	return srv.secrets.Purge(models.GroupScope(id))
}

// member loads a group the principal belongs to. One that cannot be found is
// indistinguishable from one the principal may not see.
func (srv *GroupService) member(principal models.Principal, id uuid.UUID) (*models.Group, error) {
	if principal.Anonymous() {
		return nil, ErrForbidden
	}

	group, err := srv.store.FindGroup(id)
	if err != nil || !group.IsMember(principal.ID) {
		return nil, ErrForbidden
	}

	return group, nil
}

func (srv *GroupService) owned(principal models.Principal, id uuid.UUID) (*models.Group, error) {
	group, err := srv.member(principal, id)
	if err != nil {
		return nil, err
	}
	if group.Owner != principal.ID {
		return nil, ErrForbidden
	}

	return group, nil
}
