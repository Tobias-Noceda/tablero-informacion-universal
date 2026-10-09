package models

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

// Group is a set of users that owns secrets together. Its members may bind
// them on any board they belong to; only the owner manages them.
type Group struct {
	Id      uuid.UUID `bson:"_id" json:"id"`
	Name    string    `bson:"name" json:"name"`
	Owner   string    `bson:"owner" json:"owner"`
	Members []string  `bson:"members" json:"members"`
	// Org is the organization the group belongs to, if any.
	Org       *uuid.UUID `bson:"org,omitempty" json:"org,omitempty"`
	CreatedAt time.Time  `bson:"createdat" json:"created_at"`

	// Role is the caller's role, filled in per response and never stored.
	Role GroupRole `bson:"-" json:"role,omitempty"`
}

// GroupRole is what someone is in a group: its owner manages the group's
// secrets, members may use them, viewers (the members of the group's
// organization) only see that the group exists.
type GroupRole string

const (
	GroupOwner  GroupRole = "owner"
	GroupMember GroupRole = "member"
	GroupViewer GroupRole = "viewer"
)

var groupRank = map[GroupRole]int{GroupViewer: 1, GroupMember: 2, GroupOwner: 3}

// AtLeast reports whether r grants everything min does. No role grants nothing.
func (r GroupRole) AtLeast(min GroupRole) bool {
	return groupRank[r] > 0 && groupRank[r] >= groupRank[min]
}

// IsMember tells whether the group itself lists userID, as owner or member.
func (g *Group) IsMember(userID string) bool {
	return userID != "" && (g.Owner == userID || slices.Contains(g.Members, userID))
}
