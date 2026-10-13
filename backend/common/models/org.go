package models

import (
	"time"

	"github.com/google/uuid"
)

// OrgRole is what someone is in an organization: admins own the org's boards
// and groups, members look at them.
type OrgRole string

const (
	OrgRoleAdmin  OrgRole = "admin"
	OrgRoleMember OrgRole = "member"
)

func (r OrgRole) Valid() bool {
	return r == OrgRoleAdmin || r == OrgRoleMember
}

type OrgMember struct {
	User string  `bson:"user" json:"user"`
	Role OrgRole `bson:"role" json:"role"`
}

// OrgMemberSummary is how an organization lists who is in it.
type OrgMemberSummary struct {
	User UserSummary `json:"user"`
	Role OrgRole     `json:"role"`
}

// Org is a tenant: the boards and groups that name it are reachable by its
// members without being shared one by one.
type Org struct {
	Id        uuid.UUID   `bson:"_id" json:"id"`
	Name      string      `bson:"name" json:"name"`
	Members   []OrgMember `bson:"members" json:"members"`
	CreatedAt time.Time   `bson:"createdat" json:"created_at"`

	// Role is the caller's role, filled in per response and never stored.
	Role OrgRole `bson:"-" json:"role,omitempty"`
}

// OrgDetail is one organization with who is in it.
type OrgDetail struct {
	Id        uuid.UUID          `json:"id"`
	Name      string             `json:"name"`
	Role      OrgRole            `json:"role"`
	Members   []OrgMemberSummary `json:"members"`
	CreatedAt time.Time          `json:"created_at"`
}

func (o *Org) RoleOf(userID string) OrgRole {
	if userID == "" {
		return ""
	}
	for _, member := range o.Members {
		if member.User == userID {
			return member.Role
		}
	}
	return ""
}

func (o *Org) Admins() int {
	admins := 0
	for _, member := range o.Members {
		if member.Role == OrgRoleAdmin {
			admins++
		}
	}
	return admins
}

func (o *Org) UserIDs() []string {
	users := make([]string, 0, len(o.Members))
	for _, member := range o.Members {
		users = append(users, member.User)
	}
	return users
}
