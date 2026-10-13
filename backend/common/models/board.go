package models

import "github.com/google/uuid"

type Position struct {
	X float32 `bson:"x" json:"x"`
	Y float32 `bson:"y" json:"y"`
}

type BoardPostIt struct {
	Id       uuid.UUID `bson:"id" json:"id"`
	Type     string    `bson:"type" json:"type"` // Reserved for future use
	Title    Title     `bson:"title" json:"title"`
	Position Position  `bson:"position" json:"position"`
}

type Strand struct {
	Id     uuid.UUID `bson:"id" json:"id"`
	Source uuid.UUID `bson:"source" json:"source"`
	Target uuid.UUID `bson:"target" json:"target"`
}

// BoardRole is what someone may do on a board: viewers look, editors work on
// the cards, the owner also decides who else is on it.
type BoardRole string

const (
	BoardOwner  BoardRole = "owner"
	BoardEditor BoardRole = "editor"
	BoardViewer BoardRole = "viewer"
)

var boardRank = map[BoardRole]int{BoardViewer: 1, BoardEditor: 2, BoardOwner: 3}

// AtLeast reports whether r grants everything min does. No role grants nothing.
func (r BoardRole) AtLeast(min BoardRole) bool {
	return boardRank[r] > 0 && boardRank[r] >= boardRank[min]
}

// Assignable roles are the ones an owner hands out; ownership is not one.
func (r BoardRole) Assignable() bool {
	return r == BoardEditor || r == BoardViewer
}

type BoardMember struct {
	User string    `bson:"user" json:"user"`
	Role BoardRole `bson:"role" json:"role"`
}

// BoardMemberSummary is how a board lists who is on it.
type BoardMemberSummary struct {
	User UserSummary `json:"user"`
	Role BoardRole   `json:"role"`
}

type Board struct {
	Id    uuid.UUID `bson:"_id" json:"id"`
	Name  string    `bson:"name" json:"name"`
	Owner string    `bson:"owner" json:"owner"`
	// Members are everyone else on the board; the owner is never listed here.
	Members []BoardMember `bson:"members" json:"members"`
	// Org is the organization the board belongs to, if any (phase 17).
	Org     *uuid.UUID    `bson:"org,omitempty" json:"org,omitempty"`
	PostIts []BoardPostIt `bson:"postits" json:"postits"`
	Strands []Strand      `bson:"strands" json:"strands"`
	Envs    []Envs        `bson:"envs" json:"envs"`

	// Role is the caller's role, filled in per response and never stored.
	Role BoardRole `bson:"-" json:"role,omitempty"`
}

// ExplicitRole is the role the board itself gives userID, before anything
// inherited (an organization) is considered.
func (b *Board) ExplicitRole(userID string) BoardRole {
	if userID == "" {
		return ""
	}
	if b.Owner == userID {
		return BoardOwner
	}
	for _, member := range b.Members {
		if member.User == userID {
			return member.Role
		}
	}
	return ""
}

// Everyone is the owner followed by every member.
func (b *Board) Everyone() []string {
	users := make([]string, 0, len(b.Members)+1)
	users = append(users, b.Owner)
	for _, member := range b.Members {
		users = append(users, member.User)
	}
	return users
}
