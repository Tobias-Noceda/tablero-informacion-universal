package models

import (
	"regexp"
	"time"

	"github.com/google/uuid"
)

var secretNameFormat = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,63}$`)

func ValidSecretName(name string) bool {
	return secretNameFormat.MatchString(name)
}

type SecretKind string

const (
	SecretApiKey SecretKind = "api_key"
	SecretBearer SecretKind = "bearer"
	SecretBasic  SecretKind = "basic"
	SecretOAuth2 SecretKind = "oauth2"
)

type Secret struct {
	Id         uuid.UUID  `bson:"_id" json:"id"`
	Board      uuid.UUID  `bson:"board" json:"board"`
	Name       string     `bson:"name" json:"name"`
	Kind       SecretKind `bson:"kind" json:"kind"`
	Ciphertext []byte     `bson:"ciphertext" json:"-"`
	Nonce      []byte     `bson:"nonce" json:"-"`
	KeyVersion int        `bson:"keyversion" json:"-"`
	CreatedAt  time.Time  `bson:"createdat" json:"created_at"`
	UpdatedAt  time.Time  `bson:"updatedat" json:"updated_at"`

	// Which OAuth2 flow this credential uses, and whether a user has already
	// consented. Both are configuration rather than secrets, so they live in
	// the clear and a listing does not have to decrypt anything.
	Flow       string `bson:"flow" json:"flow,omitempty"`
	Authorized bool   `bson:"authorized" json:"authorized,omitempty"`
}

type SecretMeta struct {
	Name       string     `json:"name"`
	Kind       SecretKind `json:"kind"`
	Flow       string     `json:"flow,omitempty"`
	Authorized bool       `json:"authorized"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func (s *Secret) Meta() SecretMeta {
	return SecretMeta{
		Name:       s.Name,
		Kind:       s.Kind,
		Flow:       s.Flow,
		Authorized: s.Authorized,
		CreatedAt:  s.CreatedAt,
		UpdatedAt:  s.UpdatedAt,
	}
}
