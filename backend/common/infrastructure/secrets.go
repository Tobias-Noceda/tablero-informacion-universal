package infrastructure

import (
	"errors"
	"time"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

var (
	ErrForbidden         = errors.New("Not allowed to manage these secrets")
	ErrUnknownCredential = errors.New("Unknown credential")
)

// SecretResolver is the only way a decrypted value reaches an execution. Kept
// narrow so the post-it service cannot reach the write or list operations.
type SecretResolver interface {
	// Resolve reads a scope by name with no principal: for the platform's own
	// secrets, which only a well-known definition can name.
	Resolve(scope models.SecretScope, names []string) (map[string]string, error)
	// ResolveAs reads what principal may use on board. A ref nobody stored is
	// absent; one the principal may not use fails with ErrForbidden.
	ResolveAs(principal models.Principal, board uuid.UUID, refs []models.SecretRef) (map[models.SecretRef]string, error)
	// CanBind fails with ErrUnknownCredential or ErrForbidden unless every
	// ref exists and principal may use it on board.
	CanBind(principal models.Principal, board uuid.UUID, refs []models.SecretRef) error
}

// ScopePolicy decides who may see or change the secrets of a scope. It is the
// single seam authentication plugs into.
type ScopePolicy interface {
	CanManage(principal models.Principal, scope models.SecretScope) error
	CanView(principal models.Principal, scope models.SecretScope) error
	// CanUse asks whether principal may bind secret into a card on board.
	CanUse(principal models.Principal, board uuid.UUID, secret *models.Secret) error
}

// BoardReader is the slice of Database the policy needs to answer "may this
// caller manage this board's credentials".
type BoardReader interface {
	FindBoard(id uuid.UUID) (*models.Board, error)
}

// TokenClient talks to an OAuth2 provider's token endpoint. Both methods write
// the result back into material rather than returning it, because a refresh
// can also replace the refresh token itself.
type TokenClient interface {
	Fetch(material *models.OAuth2Material) error
	Exchange(material *models.OAuth2Material, code, redirectURI, verifier string) error
}

// HandshakeStore holds an in-flight OAuth2 authorization between the redirect
// out and the callback back. Take must return an entry at most once, so a
// replayed state cannot drive a second exchange.
type HandshakeStore interface {
	Put(key string, value []byte, ttl time.Duration) error
	Take(key string) ([]byte, error)
}

// Locker serialises token refreshes across processes. Two workers refreshing
// the same credential at once would both spend the refresh token, and a
// provider that rotates them invalidates whichever lands second.
type Locker interface {
	// Acquire reports whether the caller now holds the lock, along with a token
	// identifying this holder.
	Acquire(key string, ttl time.Duration) (token string, held bool, err error)
	// Release gives the lock back, and must be a no-op unless token still
	// identifies the current holder. A holder that overran the TTL would
	// otherwise delete the lock its successor is relying on.
	Release(key, token string) error
}

type SecretStore interface {
	// Creates or replaces a secret, keyed by scope + name
	UpsertSecret(secret *models.Secret) error
	// Find the named secrets of a scope. Missing names are simply absent.
	FindSecrets(scope models.SecretScope, names []string) ([]models.Secret, error)
	// Find every secret of a scope
	ListSecrets(scope models.SecretScope) ([]models.Secret, error)
	// Delete a secret by name
	DeleteSecret(scope models.SecretScope, name string) error
	// Delete every secret of a scope
	DeleteSecrets(scope models.SecretScope) error
	// SetGrants replaces who else may bind a secret. ErrUnknownCredential if
	// the secret does not exist.
	SetGrants(scope models.SecretScope, name string, grants []models.Grant) error
	// FindGranted returns every secret with a grant for any of the audiences.
	FindGranted(audiences []models.Audience) ([]models.Secret, error)
}
