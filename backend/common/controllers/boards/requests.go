package boards

import (
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

type CreateBoardRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateBoardNameRequest struct {
	Name string `json:"name" binding:"required"`
}

type SetMemberRequest struct {
	Email string           `json:"email" binding:"required"`
	Role  models.BoardRole `json:"role" binding:"required"`
}

type StrandRequest struct {
	Source uuid.UUID `json:"source" binding:"required"`
	Target uuid.UUID `json:"target" binding:"required"`
}
