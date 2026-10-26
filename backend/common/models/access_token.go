package models

import (
	"time"

	"github.com/google/uuid"
)

// AccessToken is a personal access token: a long-lived, read-only bearer a
// developer hands to a tool (an MCP client) so it acts as them. Only the
// hash of the token is stored; the token itself is shown once on creation.
type AccessToken struct {
	Id         uuid.UUID  `bson:"_id" json:"id"`
	User       string     `bson:"user" json:"-"`
	Name       string     `bson:"name" json:"name"`
	Prefix     string     `bson:"prefix" json:"prefix"`
	Hash       string     `bson:"hash" json:"-"`
	CreatedAt  time.Time  `bson:"created_at" json:"created_at"`
	ExpiresAt  time.Time  `bson:"expires_at" json:"expires_at"`
	LastUsedAt *time.Time `bson:"last_used_at,omitempty" json:"last_used_at,omitempty"`
}
