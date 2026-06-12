package models

import (
	"net/url"

	"github.com/google/uuid"
)

type Request struct {
	Method  string            `bson:"method"`
	Headers map[string]string `bson:"headers"`
	Queries map[string]string `bson:"queries"`
	Body    string            `bson:"body"`
}

type PostIts struct {
	Id        uuid.UUID         `bson:"_id"`
	Board     uuid.UUID         `bson:"board"`
	Params    map[string]string `bson:"params"` // Soon, I promise you will really shine
	WellKnown string            `bson:"wellknown"`
	Resource  *url.URL          `bson:"resource"`
	Request   Request           `bson:"request"`
	Response  string            `bson:"response"` // API response type, used in Accept header
	Query     string            `bson:"query"`    // https//github.com/itchyny/gojq
	Rate      int               `bson:"rate"`     // A rate-less post-it should only be updated on creation
	Envs      []Envs            `bson:"envs"`     // Board + Post-it defined env variables
}
