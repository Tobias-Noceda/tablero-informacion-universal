package models

import (
	"github.com/google/uuid"
)

type Position struct {
	X float32 `bson:"x"`
	Y float32 `bson:"y"`
}

type BoardPostIt struct {
	Id       uuid.UUID `bson:"id"`
	Type     string    `bson:"type"` // Reserved for future use
	Position Position  `bson:"position"`
}

type Strand struct {
	Source uuid.UUID `bson:"source"`
	Target uuid.UUID `bson:"target"`
}

type Board struct {
	Id            uuid.UUID     `bson:"_id"`
	Name          string        `bson:"name"`
	Owner         string        `bson:"owner"`
	Collaborators []string      `bson:"collaborators"`
	PostIts       []BoardPostIt `bson:"postits"`
	Strands       []Strand      `bson:"strands"`
	Envs          []Envs        `bson:"envs"`
}
