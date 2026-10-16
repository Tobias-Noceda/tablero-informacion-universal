package models

// Principal is whoever is making the request: the subject of the bearer
// token, plus the one platform-wide flag policies need without a lookup.
type Principal struct {
	ID    string
	Admin bool
}

func (p Principal) Anonymous() bool {
	return p.ID == ""
}
