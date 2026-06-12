package postits

import "github.com/google/uuid"

type CreatePostItRequest struct {
	Board     uuid.UUID         `json:"owner" binding:"required"`
	WellKnown string            `json:"well-known"`
	Params    map[string]string `json:"params"`
}

type MovePostItRequest struct {
	X float32 `json:"x" binding:"required,number"`
	Y float32 `json:"y" binding:"required,number"`
}
