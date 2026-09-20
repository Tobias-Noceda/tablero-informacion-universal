package models

import (
	"strings"

	"github.com/google/uuid"
)

type ScopeKind string

const (
	ScopeBoard  ScopeKind = "board"
	ScopeUser   ScopeKind = "user"
	ScopeMember ScopeKind = "member"
	ScopeSystem ScopeKind = "system"
)

const memberOwnerSeparator = ":"

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

// MemberScope holds what one user keeps in one board: it is purged with the
// membership, and nobody else on the board can see or bind it.
func MemberScope(board uuid.UUID, userID string) SecretScope {
	return SecretScope{Kind: ScopeMember, Owner: board.String() + memberOwnerSeparator + userID}
}

func (s SecretScope) Member() (board uuid.UUID, userID string, ok bool) {
	if s.Kind != ScopeMember {
		return uuid.Nil, "", false
	}

	rawBoard, userID, found := strings.Cut(s.Owner, memberOwnerSeparator)
	if !found || userID == "" {
		return uuid.Nil, "", false
	}

	board, err := uuid.Parse(rawBoard)
	if err != nil {
		return uuid.Nil, "", false
	}

	return board, userID, true
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
	case ScopeMember:
		_, _, ok := s.Member()
		return ok
	case ScopeSystem:
		return s.Owner == ""
	default:
		return false
	}
}
