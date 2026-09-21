package infrastructure

import (
	"errors"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

var ErrGroupNotFound = errors.New("group not found")

// GroupReader is what authorization needs: who is in a group, and which
// groups a user is in.
type GroupReader interface {
	FindGroup(id uuid.UUID) (*models.Group, error)
	FindUserGroups(userID string) ([]models.Group, error)
}

type GroupStore interface {
	GroupReader
	CreateGroup(group *models.Group) error
	DeleteGroup(id uuid.UUID) error
	AddGroupMember(id uuid.UUID, userID string) error
	RemoveGroupMember(id uuid.UUID, userID string) error
}
