package models

import (
	"time"

	"github.com/google/uuid"
)

// DataKey is the key that encrypts the secrets of one scope. It is stored
// wrapped by the master key, so a database dump alone reveals nothing, and it
// records which master key version wrapped it so rotations can be tracked.
type DataKey struct {
	Id         uuid.UUID   `bson:"_id" json:"id"`
	Scope      SecretScope `bson:"scope" json:"scope"`
	WrappedKey []byte      `bson:"wrappedkey" json:"-"`
	Nonce      []byte      `bson:"nonce" json:"-"`
	KEKVersion int         `bson:"kekversion" json:"kek_version"`
	Active     bool        `bson:"active" json:"active"`
	CreatedAt  time.Time   `bson:"createdat" json:"created_at"`
	RetiredAt  *time.Time  `bson:"retiredat,omitempty" json:"retired_at,omitempty"`
}
