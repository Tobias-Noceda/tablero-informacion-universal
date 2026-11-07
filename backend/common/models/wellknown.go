package models

import (
	"errors"
	"fmt"
	"maps"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"
)

// WellKnown is a ready-made card: the request it runs, the form that fills
// it in and what it answers with. Stored in the well_knowns collection; the
// platform's own definitions are seeded on boot (services/wellknowns).
type WellKnown struct {
	Key           string `bson:"_id" json:"key"`
	NameEn        string `bson:"name_en" json:"name_en"`
	NameEs        string `bson:"name_es" json:"name_es"`
	DescriptionEn string `bson:"description_en,omitempty" json:"description_en,omitempty"`
	DescriptionEs string `bson:"description_es,omitempty" json:"description_es,omitempty"`
	// Position in the card picker, lowest first (ties by key).
	Order int `bson:"order" json:"order"`

	// An example title for a new card, in each language, may use the outputs
	// as {{key}}. Only a suggestion: a card stores the one title it is given.
	TitleEn string `bson:"title_en" json:"title_en"`
	TitleEs string `bson:"title_es" json:"title_es"`

	// Empty for a card without a request (static_card), which answers with
	// its own params.
	Resource string            `bson:"resource" json:"resource"`
	Request  Request           `bson:"request" json:"request"`
	Response string            `bson:"response" json:"response"`
	Query    map[string]string `bson:"query" json:"query"`
	Rate     int               `bson:"rate" json:"rate"`

	// The card's variables: the form the frontend renders and the template's
	// Params (a parameter without a default must be supplied).
	Parameters []WellKnownParameter `bson:"parameters" json:"parameters"`
	Outputs    []WellKnownOutput    `bson:"outputs" json:"outputs"`

	// Platform credentials, by the lowercase placeholder the request uses.
	// Resolved from the system scope when the card runs, never stored on it.
	SystemSecrets map[string]SystemSecretName `bson:"system_secrets,omitempty" json:"system_secrets,omitempty"`

	// Builtin marks the definitions the platform seeds.
	Builtin bool `bson:"builtin" json:"builtin"`
	// Deleted is a tombstone: hidden from listings and new cards, kept so the
	// seed does not bring it back and existing cards keep their credentials.
	Deleted   bool      `bson:"deleted" json:"-"`
	UpdatedAt time.Time `bson:"updated_at,omitempty" json:"updated_at,omitzero"`
	UpdatedBy string    `bson:"updated_by,omitempty" json:"updated_by,omitempty"`

	// template marks a PostitTemplate seen through AsWellKnown, the only
	// definition allowed a custom_ key.
	template bool
}

type WellKnownParameterType string

const (
	ParameterString WellKnownParameterType = "string"
	ParameterNumber WellKnownParameterType = "number"
	// ParameterSecret renders a picker over the board's credentials.
	ParameterSecret WellKnownParameterType = "secret"
)

type WellKnownParameter struct {
	// Exact key in the card's Params, usually a "$placeholder".
	Key           string                 `bson:"key" json:"key"`
	LabelEn       string                 `bson:"label_en" json:"label_en"`
	LabelEs       string                 `bson:"label_es" json:"label_es"`
	Type          WellKnownParameterType `bson:"type" json:"type"`
	PlaceholderEn *string                `bson:"placeholder_en,omitempty" json:"placeholder_en,omitempty"`
	PlaceholderEs *string                `bson:"placeholder_es,omitempty" json:"placeholder_es,omitempty"`
	// Prefilled value. A parameter without one must be supplied by the user.
	Default *string `bson:"default,omitempty" json:"default,omitempty"`
}

type WellKnownOutput struct {
	// A key of the card's response, usable in its title as {{key}}.
	Key     string `bson:"key" json:"key"`
	LabelEn string `bson:"label_en" json:"label_en"`
	LabelEs string `bson:"label_es" json:"label_es"`
	// Sample values, only to preview a title before the card exists.
	ExampleEn any `bson:"example_en" json:"example_en"`
	ExampleEs any `bson:"example_es" json:"example_es"`
}

var (
	ErrWellKnownMissingVariable = errors.New("Missing required variable")

	wellKnownKey = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
	placeholder  = regexp.MustCompile(`^\$[a-z][a-z0-9_]*$`)
)

// Instantiate builds the post-it a new card stores, with params overriding
// the parameters' defaults. The title is the English example, for a caller
// that sends none (the frontend always does).
func (wk WellKnown) Instantiate(params map[string]string) (*PostIts, *BoardPostItTitle, error) {
	postit := &PostIts{
		WellKnown: wk.Key,
		Request: Request{
			Method:  wk.Request.Method,
			Headers: maps.Clone(wk.Request.Headers),
			Queries: maps.Clone(wk.Request.Queries),
			Body:    wk.Request.Body,
		},
		Response: wk.Response,
		Query:    maps.Clone(wk.Query),
		Rate:     wk.Rate,
	}

	if wk.Resource != "" {
		resource, err := url.Parse(wk.Resource)
		if err != nil {
			return nil, nil, err
		}
		postit.Resource = resource
	}

	if len(wk.Parameters) > 0 {
		postit.Params = make(map[string]string, len(wk.Parameters))
	}
	for _, p := range wk.Parameters {
		if value, ok := params[p.Key]; ok {
			postit.Params[p.Key] = value
			continue
		}
		if p.Default != nil && *p.Default != "" {
			postit.Params[p.Key] = *p.Default
			continue
		}
		return nil, nil, ErrWellKnownMissingVariable
	}

	return postit, &BoardPostItTitle{Text: wk.TitleEn}, nil
}

// Validate checks a definition is consistent before it is stored: every
// placeholder the request uses is declared, the outputs are what the query
// produces, and platform credentials are ones the code knows.
func (wk WellKnown) Validate() error {
	if !wellKnownKey.MatchString(wk.Key) {
		return fmt.Errorf("key must match %s", wellKnownKey)
	}
	if IsPostitTemplateKey(wk.Key) && !wk.template {
		return fmt.Errorf("keys starting with %q are reserved for post-it templates", PostitTemplatePrefix)
	}
	if strings.TrimSpace(wk.NameEn) == "" || strings.TrimSpace(wk.NameEs) == "" {
		return errors.New("name_en and name_es are required")
	}
	if strings.TrimSpace(wk.TitleEn) == "" || strings.TrimSpace(wk.TitleEs) == "" {
		return errors.New("title_en and title_es are required")
	}
	if wk.Rate < 0 {
		return errors.New("rate must not be negative")
	}

	params := make(map[string]bool, len(wk.Parameters))
	for _, p := range wk.Parameters {
		if p.Key == "" || params[p.Key] {
			return fmt.Errorf("parameter %q is empty or repeated", p.Key)
		}
		if strings.HasPrefix(p.Key, "$") && !placeholder.MatchString(p.Key) {
			return fmt.Errorf("parameter %q: placeholders are lowercase ($name), uppercase names are user secrets", p.Key)
		}
		if p.LabelEn == "" || p.LabelEs == "" {
			return fmt.Errorf("parameter %q needs label_en and label_es", p.Key)
		}
		switch p.Type {
		case ParameterString, ParameterNumber, ParameterSecret:
		default:
			return fmt.Errorf("parameter %q: type must be string, number or secret", p.Key)
		}
		params[p.Key] = true
	}

	for name, secret := range wk.SystemSecrets {
		if !placeholder.MatchString(name) {
			return fmt.Errorf("system secret placeholder %q must be lowercase ($name)", name)
		}
		if params[name] {
			return fmt.Errorf("%q is both a parameter and a system secret", name)
		}
		if !slices.Contains(KnownSystemSecrets, secret) {
			return fmt.Errorf("system secret %q is unknown", secret)
		}
	}

	outputs := make(map[string]bool, len(wk.Outputs))
	for _, o := range wk.Outputs {
		if o.Key == "" || outputs[o.Key] {
			return fmt.Errorf("output %q is empty or repeated", o.Key)
		}
		if o.LabelEn == "" || o.LabelEs == "" {
			return fmt.Errorf("output %q needs label_en and label_es", o.Key)
		}
		outputs[o.Key] = true
	}

	// A card without a request answers with its own params.
	if wk.Resource == "" {
		if len(wk.SystemSecrets) > 0 || len(wk.Query) > 0 {
			return errors.New("a well-known without a resource has no query nor system secrets")
		}
		for key := range outputs {
			if !params[key] {
				return fmt.Errorf("output %q is not a parameter", key)
			}
		}
		return nil
	}

	resource, err := url.Parse(wk.Resource)
	if err != nil || (resource.Scheme != "http" && resource.Scheme != "https") || resource.Host == "" {
		return errors.New("resource must be an http(s) URL")
	}
	if wk.Response != "json" && wk.Response != "html" {
		return errors.New("response must be json or html")
	}
	if wk.Request.Method != "GET" {
		return errors.New("request method must be GET")
	}
	if len(wk.Query) == 0 {
		return errors.New("query is required")
	}
	if len(outputs) != len(wk.Query) {
		return errors.New("outputs must list exactly the query keys")
	}
	for key := range wk.Query {
		if !outputs[key] {
			return fmt.Errorf("query key %q has no output", key)
		}
	}

	for _, source := range []map[string]string{wk.Request.Headers, wk.Request.Queries} {
		for field, value := range source {
			if !strings.HasPrefix(value, "$") {
				continue
			}
			if !placeholder.MatchString(value) {
				return fmt.Errorf("%s: %q must be a lowercase placeholder", field, value)
			}
			if _, system := wk.SystemSecrets[value]; !params[value] && !system {
				return fmt.Errorf("%s: %q is neither a parameter nor a system secret", field, value)
			}
		}
	}

	return nil
}
