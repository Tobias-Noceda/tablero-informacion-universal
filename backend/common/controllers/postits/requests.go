package postits

import (
	"net/url"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

type CreatePostItRequest struct {
	Board     uuid.UUID                   `json:"board" binding:"required"`
	WellKnown string                      `json:"well_known"`
	Params    map[string]string           `json:"params" binding:"dive,keys,required,endkeys,required"`
	Title     *models.BoardPostItTitle    `json:"title" binding:"required_without=WellKnown"`
	Resource  *url.URL                    `json:"resource" binding:"required_without=WellKnown"`
	Request   models.Request              `json:"request" binding:"required_without=WellKnown"`
	Response  string                      `json:"response" binding:"required_without=WellKnown,omitempty,eq=json|eq=html"`
	Query     map[string]string           `json:"query" binding:"required_without=WellKnown,omitempty,gt=0,dive,keys,required,endkeys,required"`
	Rate      int                         `json:"rate" binding:"required_without=WellKnown,omitempty,gt=5"`
	Envs      []models.Envs               `json:"envs"`
	Bindings  map[string]models.SecretRef `json:"bindings"`
}

type MovePostItRequest struct {
	// Pointers so 0 is a position: `required` refuses a zero value.
	X *float32 `json:"x" binding:"required"`
	Y *float32 `json:"y" binding:"required"`
}

type UpdatePostItRequest struct {
	Params   map[string]string           `json:"params"`
	Bindings map[string]models.SecretRef `json:"bindings"`
	Query    map[string]string           `json:"query"`
	Response *string                     `json:"response"`
	Rate     *int                        `json:"rate"`
}

type UpdatePostItTitleRequest struct {
	Title models.BoardPostItTitle `json:"title" binding:"required"`
}
