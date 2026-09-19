package models

import "github.com/google/uuid"

type ScopeKind string

const (
	ScopeBoard  ScopeKind = "board"
	ScopeUser   ScopeKind = "user"
	ScopeSystem ScopeKind = "system"
)

// SecretScope is who a secret belongs to. Everything that isolates secrets
// from each other (encryption key, AAD, locks, listings) is keyed by it.
type SecretScope struct {
	Kind  ScopeKind `bson:"kind" json:"kind"`
	Owner string    `bson:"owner" json:"owner,omitempty"`
}

var SystemScope = SecretScope{Kind: ScopeSystem}

func BoardScope(id uuid.UUID) SecretScope {
	return SecretScope{Kind: ScopeBoard, Owner: id.String()}
}

func UserScope(userID string) SecretScope {
	return SecretScope{Kind: ScopeUser, Owner: userID}
}

func (s SecretScope) Key() string {
	if s.Kind == ScopeSystem {
		return string(ScopeSystem)
	}

	return string(s.Kind) + ":" + s.Owner
}

func (s SecretScope) Valid() bool {
	switch s.Kind {
	case ScopeBoard:
		_, err := uuid.Parse(s.Owner)
		return err == nil
	case ScopeUser:
		return s.Owner != ""
	case ScopeSystem:
		return s.Owner == ""
	default:
		return false
	}
}
