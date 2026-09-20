package postits

import (
	"net/url"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

type CreatePostItRequest struct {
	Board     uuid.UUID         `json:"board" binding:"required"`
	WellKnown string            `json:"well_known"`
	Params    map[string]string `json:"params" binding:"dive,keys,required,endkeys,required"`
	Title     models.Title      `json:"title" binding:"required_without=WellKnown"`
	Resource  *url.URL          `json:"resource" binding:"required_without=WellKnown"`
	Request   models.Request    `json:"request"`
	Response  string            `json:"response" binding:"required_without=WellKnown,eq=json|eq=html"`
	Query     map[string]string `json:"query" binding:"required_without=WellKnown,gt=0,dive,keys,required,endkeys,required"`
	Rate      int               `json:"rate" binding:"required_without=WellKnown,gt=5"`
	Envs      []models.Envs     `json:"envs"`
}

type MovePostItRequest struct {
	X float32 `json:"x" binding:"required,number"`
	Y float32 `json:"y" binding:"required,number"`
}

type UpdatePostItRequest struct {
	Title    *models.Title     `json:"title"`
	Params   map[string]string `json:"params"`
	Query    map[string]string `json:"query"`
	Response *string           `json:"response"`
	Rate     *int              `json:"rate"`
}
