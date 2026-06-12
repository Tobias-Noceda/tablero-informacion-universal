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
	Response  string            `bson:"response" json:"response"` // API response type, used in Accept header
	Query     string            `bson:"query" json:"query"`    // https//github.com/itchyny/gojq
	Rate      int               `bson:"rate" json:"rate"`     // A rate-less post-it should only be updated on creation
	Envs      []Envs            `bson:"envs" json:"envs"`     // Board + Post-it defined env variables
}
