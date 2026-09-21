package models

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

// Group is a set of users that owns secrets together. Its members may bind
// them on any board they belong to; only the owner manages them.
type Group struct {
	Id        uuid.UUID `bson:"_id" json:"id"`
	Name      string    `bson:"name" json:"name"`
	Owner     string    `bson:"owner" json:"owner"`
	Members   []string  `bson:"members" json:"members"`
	CreatedAt time.Time `bson:"createdat" json:"created_at"`
}

func (g *Group) IsMember(userID string) bool {
	return userID != "" && (g.Owner == userID || slices.Contains(g.Members, userID))
}
