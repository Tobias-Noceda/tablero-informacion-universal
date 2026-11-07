package infrastructure

import (
	"errors"

	"github.com/Secreto31126/tesis/common/models"
)

var ErrPostitTemplateNotFound = errors.New("post-it template not found")

// PostitTemplateStore keeps the card types users define for themselves, a
// board or an organization. Deleting a board or an organization removes its
// own (the Database does it with the board or org).
type PostitTemplateStore interface {
	FindPostitTemplate(key string) (*models.PostitTemplate, error)
	// FindPostitTemplates returns the stored ones among keys, in no order.
	FindPostitTemplates(keys []string) ([]models.PostitTemplate, error)
	// ListPostitTemplates returns those of any of scopes, by name.
	ListPostitTemplates(scopes ...models.PostitTemplateScope) ([]models.PostitTemplate, error)
	CountPostitTemplates(scope models.PostitTemplateScope) (int64, error)
	// PutPostitTemplate creates or replaces one by key.
	PutPostitTemplate(c *models.PostitTemplate) error
	DeletePostitTemplate(key string) error
}
