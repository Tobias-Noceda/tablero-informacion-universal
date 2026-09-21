package models

// Principal is whoever is making the request. Until authentication exists it
// is the cognito_id the request carries; afterwards, the subject of the token.
type Principal struct {
	ID string
}

func (p Principal) Anonymous() bool {
	return p.ID == ""
}
