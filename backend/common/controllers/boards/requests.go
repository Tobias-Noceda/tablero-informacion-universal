package boards

import "github.com/google/uuid"

type CreateBoardRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateBoardNameRequest struct {
	Name string `json:"name" binding:"required"`
}

type CollaboratorRequest struct {
	User string `json:"user" binding:"required"`
}

type StrandRequest struct {
	Source uuid.UUID `json:"source" binding:"required"`
	Target uuid.UUID `json:"target" binding:"required"`
}
