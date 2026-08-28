package models

import (
	"github.com/google/uuid"
)

type Position struct {
	X float32 `bson:"x" json:"x"`
	Y float32 `bson:"y" json:"y"`
}

type BoardPostIt struct {
	Id       uuid.UUID `bson:"id" json:"id"`
	Type     string    `bson:"type" json:"type"` // Reserved for future use
	Position Position  `bson:"position" json:"position"`
}

type Strand struct {
	Id     uuid.UUID `bson:"id" json:"id"`
	Source uuid.UUID `bson:"source" json:"source"`
	Target uuid.UUID `bson:"target" json:"target"`
}

type Board struct {
	Id            uuid.UUID     `bson:"_id" json:"id"`
	Name          string        `bson:"name" json:"name"`
	Owner         string        `bson:"owner" json:"owner"`
	Collaborators []string      `bson:"collaborators" json:"collaborators"`
	PostIts       []BoardPostIt `bson:"postits" json:"postits"`
	Strands       []Strand      `bson:"strands" json:"strands"`
	Envs          []Envs        `bson:"envs" json:"envs"`
}
