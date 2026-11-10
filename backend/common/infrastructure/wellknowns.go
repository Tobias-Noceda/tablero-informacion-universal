package infrastructure

import (
	"errors"

	"github.com/Secreto31126/tesis/common/models"
)

var (
	ErrWellKnownNotFound   = errors.New("well-known not found")
	ErrWellKnownNotDeleted = errors.New("well-known is not deleted")
)

// WellKnownReader is what running cards needs. FindWellKnown also returns
// tombstones (Deleted): cards made before a deletion keep working.
type WellKnownReader interface {
	FindWellKnown(key string) (*models.WellKnown, error)
}

type WellKnownStore interface {
	WellKnownReader
	// ListWellKnowns returns the live definitions, by order then key.
	ListWellKnowns() ([]models.WellKnown, error)
	// PutWellKnown creates or replaces a definition, reviving a tombstone.
	PutWellKnown(wk *models.WellKnown) error
	// DeleteWellKnown tombstones a live definition.
	DeleteWellKnown(key string) error
	// RestoreWellKnown revives a tombstone as it was.
	RestoreWellKnown(key string) error
	// SeedWellKnowns inserts the definitions whose key is not stored at all:
	// an edited or deleted one is left alone.
	SeedWellKnowns(defs []models.WellKnown) error
}
