package postits

import (
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

type CreatePostItRequest struct {
	CognitoID string                      `json:"cognito_id" binding:"required"`
	Board     uuid.UUID                   `json:"board" binding:"required"`
	WellKnown string                      `json:"well_known"`
	Params    map[string]string           `json:"params"`
	Bindings  map[string]models.SecretRef `json:"bindings"`
}

type MovePostItRequest struct {
	X float32 `json:"x" binding:"required,number"`
	Y float32 `json:"y" binding:"required,number"`
}

type UpdatePostItRequest struct {
	CognitoID string                      `json:"cognito_id" binding:"required"`
	Title     *models.Title               `json:"title"`
	Params    map[string]string           `json:"params"`
	Bindings  map[string]models.SecretRef `json:"bindings"`
	Query     map[string]string           `json:"query"`
	Response  *string                     `json:"response"`
	Rate      *int                        `json:"rate"`
}
