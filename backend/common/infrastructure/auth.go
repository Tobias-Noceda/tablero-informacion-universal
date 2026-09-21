package infrastructure

import (
	"errors"
	"time"

	"github.com/Secreto31126/tesis/common/models"
)

var (
	ErrInvalidToken    = errors.New("invalid token")
	ErrSessionNotFound = errors.New("session not found")
	ErrRateLimited     = errors.New("too many attempts")
)

type Claims struct {
	Subject   string
	Admin     bool
	IssuedAt  time.Time
	ExpiresAt time.Time
	ID        string
}

type TokenSigner interface {
	Sign(claims Claims) (string, error)
}

type TokenVerifier interface {
	Verify(token string) (Claims, error)
}

// Session is one refresh token. Tokens of the same family descend from one
// login; presenting a spent one reveals theft and takes the whole family down.
type Session struct {
	TokenHash string    `json:"token_hash"`
	Family    string    `json:"family"`
	UserID    string    `json:"user_id"`
	IssuedAt  time.Time `json:"issued_at"`
}

type SessionStore interface {
	PutSession(session Session, ttl time.Duration) error
	TakeSession(tokenHash string) (*Session, error)
	MarkSpent(tokenHash, family string, at time.Time, ttl time.Duration) error
	Spent(tokenHash string) (family string, at time.Time, found bool, err error)
	RevokeFamily(family string) error
	RevokeUser(userID string) error
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(encoded, password string) (bool, error)
}

type RateLimiter interface {
	Allow(key string, limit int, window time.Duration) (bool, error)
}

type Mailer interface {
	Send(mail models.Mail) error
}

// ExternalIdentity is what an identity provider vouches for after a
// successful sign-in.
type ExternalIdentity struct {
	Provider      models.IdentityProvider
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
	Picture       string
}

type IdentityProvider interface {
	AuthURL(state, nonce, codeChallenge, redirectURI string) string
	Complete(code, redirectURI, verifier, nonce string) (*ExternalIdentity, error)
}
