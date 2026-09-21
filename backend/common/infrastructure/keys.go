package infrastructure

import (
	"errors"
	"time"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

var (
	ErrKeyNotFound     = errors.New("data key not found")
	ErrActiveKeyExists = errors.New("scope already has an active data key")
)

// KeyStore persists wrapped data keys. At most one key per scope may be
// active; InsertKey must fail with ErrActiveKeyExists rather than allow a
// second, so two workers creating a scope's first key converge on one.
type KeyStore interface {
	FindActiveKey(scope models.SecretScope) (*models.DataKey, error)
	FindKey(id uuid.UUID) (*models.DataKey, error)
	InsertKey(key *models.DataKey) error
	RetireKey(id uuid.UUID, at time.Time) error
	// Rewrap replaces the wrapping of a key without touching anything else.
	Rewrap(id uuid.UUID, wrapped, nonce []byte, kekVersion int) error
	DeleteRetiredKeys(scope models.SecretScope) error
	DeleteKeys(scope models.SecretScope) error
	ListKeys() ([]models.DataKey, error)
}

// ScopePurger removes everything a scope owns in the vault. Deleting the
// scope's data key is what makes the removal final.
type ScopePurger interface {
	Purge(scope models.SecretScope) error
}
