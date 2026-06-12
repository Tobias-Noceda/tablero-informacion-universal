package boards

import "github.com/google/uuid"

type CreateBoardRequest struct {
	Name  string `json:"name" binding:"required"`
	Owner string `json:"owner" binding:"required"`
}

type UpdateBoardNameRequest struct {
	Name string `json:"name" binding:"required"`
}

type CollaboratorRequest struct {
	CognitoID string `json:"cognito_id" binding:"required"`
}

type StrandRequest struct {
	Source uuid.UUID `json:"source" binding:"required"`
	Target uuid.UUID `json:"target" binding:"required"`
}
