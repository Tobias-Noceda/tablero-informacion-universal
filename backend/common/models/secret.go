package models

import (
	"encoding/base64"
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
	// The platform's own application at a provider. System scope only.
	SecretOAuth2Client SecretKind = "oauth2_client"
)

type Secret struct {
	Id         uuid.UUID   `bson:"_id" json:"id"`
	Scope      SecretScope `bson:"scope" json:"scope"`
	Name       string      `bson:"name" json:"name"`
	Kind       SecretKind  `bson:"kind" json:"kind"`
	Ciphertext []byte      `bson:"ciphertext" json:"-"`
	Nonce      []byte      `bson:"nonce" json:"-"`
	KeyID      uuid.UUID   `bson:"keyid" json:"-"`
	CreatedAt  time.Time   `bson:"createdat" json:"created_at"`
	UpdatedAt  time.Time   `bson:"updatedat" json:"updated_at"`

	// Which OAuth2 flow this credential uses, and whether a user has already
	// consented. Both are configuration rather than secrets, so they live in
	// the clear and a listing does not have to decrypt anything.
	Flow       string        `bson:"flow" json:"flow,omitempty"`
	Authorized bool          `bson:"authorized" json:"authorized,omitempty"`
	Provider   OAuthProvider `bson:"provider" json:"provider,omitempty"`
}

// SecretRef is how a post-it names a secret outside its board: the pair the
// store keys on, never the value.
type SecretRef struct {
	Scope SecretScope `bson:"scope" json:"scope"`
	Name  string      `bson:"name" json:"name"`
}

type SecretMeta struct {
	Scope      SecretScope   `json:"scope"`
	Name       string        `json:"name"`
	Kind       SecretKind    `json:"kind"`
	Flow       string        `json:"flow,omitempty"`
	Authorized bool          `json:"authorized"`
	Provider   OAuthProvider `json:"provider,omitempty"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
}

func Present(kind SecretKind, value string) string {
	switch kind {
	case SecretBearer:
		return "Bearer " + value
	case SecretBasic:
		// The stored value is "user:password"; the wire form is base64.
		return "Basic " + base64.StdEncoding.EncodeToString([]byte(value))
	default:
		return value
	}
}

func (s *Secret) Meta() SecretMeta {
	return SecretMeta{
		Scope:      s.Scope,
		Name:       s.Name,
		Kind:       s.Kind,
		Flow:       s.Flow,
		Authorized: s.Authorized,
		Provider:   s.Provider,
		CreatedAt:  s.CreatedAt,
		UpdatedAt:  s.UpdatedAt,
	}
}
