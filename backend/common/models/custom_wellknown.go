package models

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// CustomWellKnownScopeKind is who a custom card type belongs to.
type CustomWellKnownScopeKind string

const (
	// CustomScopeUser: only its owner (a user id) sees and uses it, on any
	// board they edit.
	CustomScopeUser CustomWellKnownScopeKind = "user"
	// CustomScopeBoard: everyone on the board (a board id) sees it, its
	// editors use and manage it.
	CustomScopeBoard CustomWellKnownScopeKind = "board"
	// CustomScopeOrg: every member of the organization (an org id) sees it and
	// uses it on the org's boards, its admins manage it.
	CustomScopeOrg CustomWellKnownScopeKind = "org"
)

type CustomWellKnownScope struct {
	Kind  CustomWellKnownScopeKind `bson:"kind" json:"kind"`
	Owner string                   `bson:"owner" json:"owner"`
}

func (s CustomWellKnownScope) Valid() bool {
	switch s.Kind {
	case CustomScopeUser, CustomScopeBoard, CustomScopeOrg:
		return s.Owner != ""
	}
	return false
}

// CustomWellKnownPrefix starts every custom key, so a custom card type can
// never be taken for one of the platform's catalogue.
const CustomWellKnownPrefix = "custom_"

const (
	maxCustomName       = 64
	maxCustomParameters = 20
	maxCustomOutputs    = 20
)

// NewCustomWellKnownKey is a fresh key for a custom card type.
func NewCustomWellKnownKey() string {
	return CustomWellKnownPrefix + strings.ReplaceAll(uuid.NewString(), "-", "")
}

func IsCustomWellKnownKey(key string) bool {
	return strings.HasPrefix(key, CustomWellKnownPrefix)
}

// CustomWellKnown is a card type a user defines for themselves, a board or an
// organization. Unlike the catalogue it is written in one language, whichever
// its author picks. Stored in custom_well_knowns.
type CustomWellKnown struct {
	Key         string               `bson:"_id" json:"key"`
	Scope       CustomWellKnownScope `bson:"scope" json:"scope"`
	Name        string               `bson:"name" json:"name"`
	Description string               `bson:"description,omitempty" json:"description,omitempty"`

	// Default title of a new card, may use the outputs as {{key}}.
	Title BoardPostItTitle `bson:"title" json:"title"`

	Resource string            `bson:"resource" json:"resource"`
	Request  Request           `bson:"request" json:"request"`
	Response string            `bson:"response" json:"response"`
	Query    map[string]string `bson:"query" json:"query"`
	Rate     int               `bson:"rate" json:"rate"`

	Parameters []CustomWellKnownParameter `bson:"parameters" json:"parameters"`
	Outputs    []CustomWellKnownOutput    `bson:"outputs" json:"outputs"`

	CreatedBy string    `bson:"created_by" json:"created_by"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedBy string    `bson:"updated_by" json:"updated_by"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

type CustomWellKnownParameter struct {
	Key         string                 `bson:"key" json:"key"`
	Label       string                 `bson:"label" json:"label"`
	Type        WellKnownParameterType `bson:"type" json:"type"`
	Placeholder *string                `bson:"placeholder,omitempty" json:"placeholder,omitempty"`
	Default     *string                `bson:"default,omitempty" json:"default,omitempty"`
}

type CustomWellKnownOutput struct {
	Key     string `bson:"key" json:"key"`
	Label   string `bson:"label" json:"label"`
	Example any    `bson:"example,omitempty" json:"example,omitempty"`
}

// AsWellKnown is the definition in the catalogue's shape, with its one
// language in both, so it instantiates and validates like any other. A
// custom card type never uses platform credentials.
func (c CustomWellKnown) AsWellKnown() WellKnown {
	wk := WellKnown{
		Key:           c.Key,
		NameEn:        c.Name,
		NameEs:        c.Name,
		DescriptionEn: c.Description,
		DescriptionEs: c.Description,
		TitleEn:       c.Title.Text,
		TitleEs:       c.Title.Text,
		Resource:      c.Resource,
		Request:       c.Request,
		Response:      c.Response,
		Query:         c.Query,
		Rate:          c.Rate,
		Parameters:    make([]WellKnownParameter, 0, len(c.Parameters)),
		Outputs:       make([]WellKnownOutput, 0, len(c.Outputs)),
		custom:        true,
	}
	for _, p := range c.Parameters {
		wk.Parameters = append(wk.Parameters, WellKnownParameter{
			Key:           p.Key,
			LabelEn:       p.Label,
			LabelEs:       p.Label,
			Type:          p.Type,
			PlaceholderEn: p.Placeholder,
			PlaceholderEs: p.Placeholder,
			Default:       p.Default,
		})
	}
	for _, o := range c.Outputs {
		wk.Outputs = append(wk.Outputs, WellKnownOutput{
			Key:       o.Key,
			LabelEn:   o.Label,
			LabelEs:   o.Label,
			ExampleEn: o.Example,
			ExampleEs: o.Example,
		})
	}
	return wk
}

// Validate applies the catalogue's rules plus a custom type's own: it always
// fetches something and stays small.
func (c CustomWellKnown) Validate() error {
	if !IsCustomWellKnownKey(c.Key) {
		return errors.New("key must be generated by the server")
	}
	if !c.Scope.Valid() {
		return errors.New("scope must be user, board or org")
	}
	if utf8.RuneCountInString(c.Name) > maxCustomName {
		return fmt.Errorf("name must be at most %d characters", maxCustomName)
	}
	if strings.TrimSpace(c.Resource) == "" {
		return errors.New("resource is required")
	}
	if len(c.Parameters) > maxCustomParameters {
		return fmt.Errorf("at most %d parameters", maxCustomParameters)
	}
	if len(c.Outputs) > maxCustomOutputs {
		return fmt.Errorf("at most %d outputs", maxCustomOutputs)
	}

	err := c.AsWellKnown().Validate()
	if err == nil {
		return nil
	}
	// The catalogue speaks of its en/es fields; a custom type has one.
	message := strings.NewReplacer(
		"name_en and name_es are", "name is",
		"title_en and title_es are", "title is",
		"needs label_en and label_es", "needs a label",
		" nor a system secret", "",
		"neither a parameter", "not a parameter",
	).Replace(err.Error())
	return errors.New(message)
}
