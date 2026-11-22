package postits

import (
	"time"

	"github.com/Secreto31126/tesis/common/models"
)

// Each preview reaches the card's API, so an editor gets previewLimit of them
// every previewWindow, across boards.
const (
	previewLimit  = 10
	previewWindow = time.Minute
)

// previewWhole is the output a preview without outputs asks for: the whole
// response, so the editor can pick what to show.
const previewWhole = "response"

// Preview is what a card would show: its outputs, or the whole response
// when the request has no outputs yet.
type Preview struct {
	Outputs  map[string]any `json:"outputs,omitempty"`
	Response any            `json:"response,omitempty"`
}

// PreviewPostIt runs a one-off card's request as the editor who is writing
// it, without saving the card or caching what it answered. It is checked
// like a card being created (the request, the credentials it binds) and goes
// out through the same SSRF-safe executer. Without outputs it answers the
// whole JSON response.
func (srv *PostItsService) PreviewPostIt(principal models.Principal, postIt *models.PostIts) (Preview, error) {
	if _, err := srv.require(principal, postIt.Board, models.BoardEditor); err != nil {
		return Preview{}, err
	}

	whole := len(postIt.Query) == 0
	if whole {
		if postIt.Response != "" && postIt.Response != "json" {
			return Preview{}, invalidOneOff("a preview without outputs needs a JSON response")
		}
		postIt.Response = "json"
		postIt.Query = map[string]string{previewWhole: "."}
	}
	if err := validateOneOff(postIt); err != nil {
		return Preview{}, err
	}

	allowed, err := srv.limiter.Allow("preview:"+principal.ID, previewLimit, previewWindow)
	if err != nil {
		return Preview{}, err
	}
	if !allowed {
		return Preview{}, ErrRateLimited
	}

	if err := srv.bind(principal, postIt.Board, postIt.Bindings); err != nil {
		return Preview{}, err
	}
	postIt.RunAs = principal.ID

	prepared, err := srv.prepare(postIt)
	if err != nil {
		return Preview{}, err
	}
	data, err := srv.exec.Execute(prepared)
	if err != nil {
		return Preview{}, err
	}

	outputs, _ := data.(map[string]any)
	if whole {
		return Preview{Response: outputs[previewWhole]}, nil
	}
	return Preview{Outputs: outputs}, nil
}
