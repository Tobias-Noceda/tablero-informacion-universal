package models

type Envs struct {
	Name string `bson:"name" json:"name"`
	Key  string `bson:"key" json:"key"`
}
