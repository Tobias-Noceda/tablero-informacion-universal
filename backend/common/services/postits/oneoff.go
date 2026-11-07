package postits

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/Secreto31126/tesis/common/models"
)

var (
	// ErrInvalidOneOff wraps why a card without a card type was refused.
	ErrInvalidOneOff = errors.New("Invalid card request")
	// ErrTemplateFixed: a card made from a card type runs that type's request.
	ErrTemplateFixed = errors.New("A card type's request can't be changed")
)

const (
	maxOneOffHeaders = 20
	maxOneOffQueries = 20
	maxOneOffOutputs = 20
)

func invalidOneOff(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidOneOff, fmt.Sprintf(format, args...))
}

// validateOneOff checks a card that carries its own request (no card type)
// the way the catalogue checks its definitions: a GET to an http(s) URL whose
// answer the query maps to outputs. It normalises the method and drops a body.
func validateOneOff(postIt *models.PostIts) error {
	resource := postIt.Resource
	if resource == nil || (resource.Scheme != "http" && resource.Scheme != "https") || resource.Host == "" {
		return invalidOneOff("resource must be an http(s) URL")
	}

	switch strings.ToUpper(postIt.Request.Method) {
	case "", "GET":
		postIt.Request.Method = "GET"
	default:
		return invalidOneOff("request method must be GET")
	}
	postIt.Request.Body = ""

	if postIt.Response != "json" && postIt.Response != "html" {
		return invalidOneOff("response must be json or html")
	}

	if len(postIt.Request.Headers) > maxOneOffHeaders {
		return invalidOneOff("at most %d headers", maxOneOffHeaders)
	}
	if len(postIt.Request.Queries) > maxOneOffQueries {
		return invalidOneOff("at most %d query parameters", maxOneOffQueries)
	}
	for _, source := range []map[string]string{postIt.Request.Headers, postIt.Request.Queries} {
		for key := range source {
			if strings.TrimSpace(key) == "" {
				return invalidOneOff("header and query names must not be empty")
			}
		}
	}

	if len(postIt.Query) == 0 {
		return invalidOneOff("query is required")
	}
	if len(postIt.Query) > maxOneOffOutputs {
		return invalidOneOff("at most %d outputs", maxOneOffOutputs)
	}
	for key, expression := range postIt.Query {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(expression) == "" {
			return invalidOneOff("every output needs a name and a query")
		}
	}

	if postIt.Rate < 0 {
		return invalidOneOff("rate must not be negative")
	}

	return nil
}

// applyOneOff merges an edit of a one-off card's request into the stored card,
// validates the result and rewrites set with what is stored.
func applyOneOff(postIt *models.PostIts, set map[string]any) error {
	_, resourceSet := set["resource"]
	_, requestSet := set["request"]
	if !resourceSet && !requestSet {
		return nil
	}
	if postIt.WellKnown != "" {
		return ErrTemplateFixed
	}

	edited := *postIt
	if raw, ok := set["resource"].(string); ok {
		resource, err := url.Parse(raw)
		if err != nil {
			return invalidOneOff("resource must be an http(s) URL")
		}
		edited.Resource = resource
	}
	if request, ok := set["request"].(models.Request); ok {
		edited.Request = request
	}
	if query, ok := set["query"].(map[string]string); ok {
		edited.Query = query
	}
	if response, ok := set["response"].(string); ok {
		edited.Response = response
	}

	if err := validateOneOff(&edited); err != nil {
		return err
	}

	if resourceSet {
		set["resource"] = edited.Resource
	}
	if requestSet {
		set["request"] = edited.Request
	}
	return nil
}
