package models

import (
	"net/url"

	"github.com/google/uuid"
)

type Request struct {
	Method  string            `bson:"method" json:"method"`
	Headers map[string]string `bson:"headers" json:"headers"`
	Queries map[string]string `bson:"queries" json:"queries"`
	Body    string            `bson:"body" json:"body"`
}

type PostIts struct {
	Id        uuid.UUID         `bson:"_id" json:"id"`
	Board     uuid.UUID         `bson:"board" json:"board"`
	Params    map[string]string `bson:"params" json:"params"` // Soon, I promise you will really shine
	WellKnown string            `bson:"wellknown" json:"wellknown"`
	Resource  *url.URL          `bson:"resource" json:"resource"`
	Request   Request           `bson:"request" json:"request"`
	Response  string            `bson:"response" json:"response"` // API response type, used to select the query parser
	Query     map[string]string `bson:"query" json:"query"`       // An object of key:query to map to, either with jq or jquery
	Rate      int               `bson:"rate" json:"rate"`         // A rate-less post-it should only be updated on creation
	Envs      []Envs            `bson:"envs" json:"envs"`         // Board + Post-it defined env variables

	// Whose credentials the card runs with: whoever last bound them. Empty
	// on cards created before this existed, which run as the board owner.
	RunAs string `bson:"runas" json:"run_as"`
	// A "$TOKEN" the card uses that does not live in the board's own scope.
	// Keyed by the token name without the dollar sign.
	Bindings map[string]SecretRef `bson:"bindings" json:"bindings"`
}
