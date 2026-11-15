package infrastructure

import (
	"errors"
	"time"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrEmailTaken   = errors.New("email already registered")
)

type UserReader interface {
	FindUser(id uuid.UUID) (*models.User, error)
	FindUserByEmail(email string) (*models.User, error)
	FindUsers(ids []string) ([]models.User, error)
}

type UserStore interface {
	UserReader
	CreateUser(user *models.User) error
	FindUserByIdentity(provider models.IdentityProvider, subject string) (*models.User, error)
	UpdateUser(id uuid.UUID, set map[string]any) error
	ReplaceIdentities(id uuid.UUID, identities []models.Identity) error
	// SetUserPlan stores the user's plan and when it lapses; nil never does.
	SetUserPlan(id uuid.UUID, plan models.Plan, expires *time.Time) error
}
