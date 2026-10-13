package postits

import (
	"net/url"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

type CreatePostItRequest struct {
	CognitoID string                      `json:"cognito_id" binding:"required"`
	Board     uuid.UUID                   `json:"board" binding:"required"`
	WellKnown string                      `json:"well_known"`
	Params    map[string]string           `json:"params" binding:"dive,keys,required,endkeys,required"`
	Title     *models.BoardPostItTitle    `json:"title" binding:"required_without=WellKnown"`
	Resource  *url.URL                    `json:"resource" binding:"required_without=WellKnown"`
	Request   models.Request              `json:"request" binding:"required_without=WellKnown"`
	Response  string                      `json:"response" binding:"required_without=WellKnown,eq=json|eq=html"`
	Query     map[string]string           `json:"query" binding:"required_without=WellKnown,gt=0,dive,keys,required,endkeys,required"`
	Rate      int                         `json:"rate" binding:"required_without=WellKnown,gt=5"`
	Envs      []models.Envs               `json:"envs"`
	Bindings  map[string]models.SecretRef `json:"bindings"`
}

type MovePostItRequest struct {
	X float32 `json:"x" binding:"required,number"`
	Y float32 `json:"y" binding:"required,number"`
}

type UpdatePostItRequest struct {
	CognitoID string                      `json:"cognito_id" binding:"required"`
	Params    map[string]string           `json:"params"`
	Bindings  map[string]models.SecretRef `json:"bindings"`
	Query     map[string]string           `json:"query"`
	Response  *string                     `json:"response"`
	Rate      *int                        `json:"rate"`
}

type UpdatePostItTitleRequest struct {
	Title models.BoardPostItTitle `json:"title" binding:"required"`
}
