package models

import "github.com/google/uuid"

type AudienceKind string

const (
	AudienceUser  AudienceKind = "user"
	AudienceGroup AudienceKind = "group"
	AudienceBoard AudienceKind = "board"
)

// Audience is who a grant reaches: one user, every member of a group, or
// every member of a board.
type Audience struct {
	Kind AudienceKind `bson:"kind" json:"kind"`
	ID   string       `bson:"id" json:"id"`
}

// Grant lets an audience bind a secret it does not own, on every board or on
// one. It is configuration, stored in the clear next to the ciphertext.
type Grant struct {
	To    Audience `bson:"to" json:"to"`
	Board string   `bson:"board,omitempty" json:"board,omitempty"`
}

func (g Grant) Valid() bool {
	if g.Board != "" && !isUUID(g.Board) {
		return false
	}

	switch g.To.Kind {
	case AudienceUser:
		return g.To.ID != ""
	case AudienceGroup:
		return isUUID(g.To.ID)
	case AudienceBoard:
		return isUUID(g.To.ID) && (g.Board == "" || g.Board == g.To.ID)
	default:
		return false
	}
}

// AppliesTo reports whether the grant is in force on the given board.
func (g Grant) AppliesTo(board uuid.UUID) bool {
	return g.Board == "" || g.Board == board.String()
}

func isUUID(raw string) bool {
	_, err := uuid.Parse(raw)
	return err == nil
}
