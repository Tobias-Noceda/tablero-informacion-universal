package infrastructure

import (
	"errors"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

var (
	ErrOrgNotFound  = errors.New("organization not found")
	ErrLastOrgAdmin = errors.New("an organization keeps at least one admin")
)

// OrgReader is what authorization needs: who is in an organization, and
// which organizations a user is in.
type OrgReader interface {
	FindOrg(id uuid.UUID) (*models.Org, error)
	FindUserOrgs(userID string) ([]models.Org, error)
}

type OrgStore interface {
	OrgReader
	CreateOrg(org *models.Org) error
	RenameOrg(id uuid.UUID, name string) error
	// SetOrgMember adds the user or changes their role. Demoting the last
	// admin fails with ErrLastOrgAdmin.
	SetOrgMember(id uuid.UUID, userID string, role models.OrgRole) error
	// RemoveOrgMember fails with ErrLastOrgAdmin for the last admin.
	RemoveOrgMember(id uuid.UUID, userID string) error
	DeleteOrg(id uuid.UUID) error
	CountOrgBoards(id uuid.UUID) (int64, error)
	CountOrgGroups(id uuid.UUID) (int64, error)
}
