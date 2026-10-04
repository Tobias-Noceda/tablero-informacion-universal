package mocks

import (
	"slices"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

// MemoryOrgStore is an OrgStore over a slice, enough for a service test.
// Boards and Groups are how many of each name an organization.
type MemoryOrgStore struct {
	Orgs   []models.Org
	Boards map[uuid.UUID]int64
	Groups map[uuid.UUID]int64
}

var _ infrastructure.OrgStore = (*MemoryOrgStore)(nil)

func (m *MemoryOrgStore) at(id uuid.UUID) *models.Org {
	for i := range m.Orgs {
		if m.Orgs[i].Id == id {
			return &m.Orgs[i]
		}
	}
	return nil
}

func (m *MemoryOrgStore) CreateOrg(org *models.Org) error {
	stored := *org
	stored.Members = slices.Clone(org.Members)
	m.Orgs = append(m.Orgs, stored)
	return nil
}

func (m *MemoryOrgStore) FindOrg(id uuid.UUID) (*models.Org, error) {
	org := m.at(id)
	if org == nil {
		return nil, infrastructure.ErrOrgNotFound
	}
	found := *org
	found.Members = slices.Clone(org.Members)
	return &found, nil
}

func (m *MemoryOrgStore) FindUserOrgs(userID string) ([]models.Org, error) {
	var out []models.Org
	for _, org := range m.Orgs {
		if org.RoleOf(userID) != "" {
			out = append(out, org)
		}
	}
	return out, nil
}

func (m *MemoryOrgStore) RenameOrg(id uuid.UUID, name string) error {
	org := m.at(id)
	if org == nil {
		return infrastructure.ErrOrgNotFound
	}
	org.Name = name
	return nil
}

func (m *MemoryOrgStore) SetOrgMember(id uuid.UUID, userID string, role models.OrgRole) error {
	org := m.at(id)
	if org == nil {
		return infrastructure.ErrOrgNotFound
	}
	for i := range org.Members {
		if org.Members[i].User == userID {
			if role != models.OrgRoleAdmin && org.Members[i].Role == models.OrgRoleAdmin && org.Admins() == 1 {
				return infrastructure.ErrLastOrgAdmin
			}
			org.Members[i].Role = role
			return nil
		}
	}
	org.Members = append(org.Members, models.OrgMember{User: userID, Role: role})
	return nil
}

func (m *MemoryOrgStore) RemoveOrgMember(id uuid.UUID, userID string) error {
	org := m.at(id)
	if org == nil {
		return infrastructure.ErrOrgNotFound
	}
	if org.RoleOf(userID) == models.OrgRoleAdmin && org.Admins() == 1 {
		return infrastructure.ErrLastOrgAdmin
	}
	org.Members = slices.DeleteFunc(org.Members, func(member models.OrgMember) bool { return member.User == userID })
	return nil
}

func (m *MemoryOrgStore) DeleteOrg(id uuid.UUID) error {
	for i := range m.Orgs {
		if m.Orgs[i].Id == id {
			m.Orgs = append(m.Orgs[:i], m.Orgs[i+1:]...)
			return nil
		}
	}
	return infrastructure.ErrOrgNotFound
}

func (m *MemoryOrgStore) CountOrgBoards(id uuid.UUID) (int64, error) {
	return m.Boards[id], nil
}

func (m *MemoryOrgStore) CountOrgGroups(id uuid.UUID) (int64, error) {
	return m.Groups[id], nil
}
