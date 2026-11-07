package infrastructure

import (
	"errors"

	"github.com/Secreto31126/tesis/common/models"
)

var ErrCustomWellKnownNotFound = errors.New("custom well-known not found")

// CustomWellKnownStore keeps the card types users define for themselves, a
// board or an organization. Deleting a board or an organization removes its
// own (the Database does it with the board or org).
type CustomWellKnownStore interface {
	FindCustomWellKnown(key string) (*models.CustomWellKnown, error)
	// FindCustomWellKnowns returns the stored ones among keys, in no order.
	FindCustomWellKnowns(keys []string) ([]models.CustomWellKnown, error)
	// ListCustomWellKnowns returns those of any of scopes, by name.
	ListCustomWellKnowns(scopes ...models.CustomWellKnownScope) ([]models.CustomWellKnown, error)
	CountCustomWellKnowns(scope models.CustomWellKnownScope) (int64, error)
	// PutCustomWellKnown creates or replaces one by key.
	PutCustomWellKnown(c *models.CustomWellKnown) error
	DeleteCustomWellKnown(key string) error
}
