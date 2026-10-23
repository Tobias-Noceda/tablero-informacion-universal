package infrastructure

import (
	"errors"
	"time"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

var ErrAccessTokenNotFound = errors.New("access token not found")

type AccessTokenStore interface {
	CreateAccessToken(token *models.AccessToken) error
	FindAccessTokenByHash(hash string) (*models.AccessToken, error)
	ListAccessTokens(user string) ([]models.AccessToken, error)
	// DeleteAccessToken removes one of user's tokens; someone else's id is
	// ErrAccessTokenNotFound.
	DeleteAccessToken(user string, id uuid.UUID) error
	// TouchAccessToken records a use, at most once a minute.
	TouchAccessToken(id uuid.UUID, at time.Time) error
}
