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
}

type SecretMeta struct {
	Name      string     `json:"name"`
	Kind      SecretKind `json:"kind"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (s *Secret) Meta() SecretMeta {
	return SecretMeta{
		Name:      s.Name,
		Kind:      s.Kind,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}
