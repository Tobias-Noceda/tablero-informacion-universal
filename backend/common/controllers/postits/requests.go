package postits

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/Secreto31126/tesis/common/models"
	srv "github.com/Secreto31126/tesis/common/services/postits"
	"github.com/google/uuid"
)

// resourceURL is a one-off card's resource as the client sends it, a URL
// string, parsed while the body is bound: a malformed URL is refused there like
// any other invalid field. Empty or absent stays the zero value, which
// `required_without=WellKnown` refuses on a one-off card; the service then
// checks it is an http(s) URL.
type resourceURL struct{ *url.URL }

func (r *resourceURL) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("%w: resource must be a string", srv.ErrInvalidOneOff)
	}
	if raw == "" {
		r.URL = nil
		return nil
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%w: resource is not a valid URL", srv.ErrInvalidOneOff)
	}
	r.URL = parsed
	return nil
}

type CreatePostItRequest struct {
	Board     uuid.UUID                   `json:"board" binding:"required"`
	WellKnown string                      `json:"well_known"`
	Params    map[string]string           `json:"params" binding:"dive,keys,required,endkeys,required"`
	Title     *models.BoardPostItTitle    `json:"title" binding:"required_without=WellKnown"`
	Resource  resourceURL                 `json:"resource" binding:"required_without=WellKnown"`
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
	// Only on a card without a card type (a one-off card).
	Resource *string         `json:"resource"`
	Request  *models.Request `json:"request"`
}

type UpdatePostItTitleRequest struct {
	Title models.BoardPostItTitle `json:"title" binding:"required"`
}
