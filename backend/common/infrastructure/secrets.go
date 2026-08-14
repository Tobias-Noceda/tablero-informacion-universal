package infrastructure

import (
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

// SecretResolver is the only way a decrypted value reaches an execution. Kept
// narrow so the post-it service cannot reach the write or list operations.
type SecretResolver interface {
	Resolve(board uuid.UUID, names []string) (map[string]string, error)
}

type SecretStore interface {
	// Creates or replaces a board secret, keyed by board + name
	UpsertSecret(secret *models.Secret) error
	// Find the named secrets of a board. Missing names are simply absent.
	FindSecrets(board uuid.UUID, names []string) ([]models.Secret, error)
	// Find every secret of a board
	ListSecrets(board uuid.UUID) ([]models.Secret, error)
	// Delete a board secret by name
	DeleteSecret(board uuid.UUID, name string) error
}
