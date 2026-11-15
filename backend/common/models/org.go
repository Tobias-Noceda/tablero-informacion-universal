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
	// Owner is who created the organization and pays for it. Read it through
	// Payer: organizations stored before it existed have none.
	Owner string `bson:"owner,omitempty" json:"owner,omitempty"`

	// Role is the caller's role, filled in per response and never stored.
	Role OrgRole `bson:"-" json:"role,omitempty"`

	// Limits is the payer's plan, which is the one the organization runs
	// on, whoever asks. SeatsUsed and BoardsUsed are what it holds against
	// them. Filled in per response and never stored.
	Limits     *Limits `bson:"-" json:"limits,omitempty"`
	SeatsUsed  *int    `bson:"-" json:"seats_used,omitempty"`
	BoardsUsed *int    `bson:"-" json:"boards_used,omitempty"`
}

// OrgDetail is one organization with who is in it.
type OrgDetail struct {
	Id        uuid.UUID          `json:"id"`
	Name      string             `json:"name"`
	Role      OrgRole            `json:"role"`
	Members   []OrgMemberSummary `json:"members"`
	CreatedAt time.Time          `json:"created_at"`
	Owner     string             `json:"owner,omitempty"`

	// The payer's plan and how much of it the organization uses.
	Limits     *Limits `json:"limits,omitempty"`
	SeatsUsed  *int    `json:"seats_used,omitempty"`
	BoardsUsed *int    `json:"boards_used,omitempty"`
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

// Payer is whose plan the organization and its boards run on: its owner, or
// the first admin of one stored before organizations had an owner.
func (o *Org) Payer() string {
	if o.Owner != "" {
		return o.Owner
	}
	for _, member := range o.Members {
		if member.Role == OrgRoleAdmin {
			return member.User
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
