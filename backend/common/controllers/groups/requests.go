package groups

import "github.com/google/uuid"

type CreateGroupRequest struct {
	Name string `json:"name" binding:"required"`
	// Org puts the group in one of the caller's organizations.
	Org *uuid.UUID `json:"org"`
}

type MemberRequest struct {
	Member string `json:"member" binding:"required"`
}
