package models

import (
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
)

type IdentityProvider string

const (
	IdentityGoogle   IdentityProvider = "google"
	IdentityPassword IdentityProvider = "password"
)

// Identity is one way a user proves who they are: a Google subject or a
// password hash. A user may hold both.
type Identity struct {
	Provider IdentityProvider `bson:"provider"`
	Subject  string           `bson:"subject,omitempty"`
	Hash     string           `bson:"hash,omitempty"`
}

type User struct {
	Id            uuid.UUID  `bson:"_id" json:"id"`
	Email         string     `bson:"email" json:"email"`
	EmailVerified bool       `bson:"emailverified" json:"email_verified"`
	Name          string     `bson:"name" json:"name"`
	Picture       string     `bson:"picture" json:"picture,omitempty"`
	Admin         bool       `bson:"admin" json:"admin"`
	Identities    []Identity `bson:"identities" json:"-"`
	CreatedAt     time.Time  `bson:"createdat" json:"created_at"`
	UpdatedAt     time.Time  `bson:"updatedat" json:"updated_at"`
}

// UserSummary is what other users get to see about someone.
type UserSummary struct {
	Id      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	Email   string    `json:"email"`
	Picture string    `json:"picture,omitempty"`
}

func (u *User) Summary() UserSummary {
	return UserSummary{Id: u.Id, Name: u.Name, Email: u.Email, Picture: u.Picture}
}

func (u *User) Principal() Principal {
	return Principal{ID: u.Id.String(), Admin: u.Admin}
}

func (u *User) Identity(provider IdentityProvider) (Identity, bool) {
	for _, identity := range u.Identities {
		if identity.Provider == provider {
			return identity, true
		}
	}
	return Identity{}, false
}

func (u *User) Providers() []IdentityProvider {
	providers := make([]IdentityProvider, 0, len(u.Identities))
	for _, identity := range u.Identities {
		providers = append(providers, identity.Provider)
	}
	return providers
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func ValidEmail(email string) bool {
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return false
	}
	_, domain, _ := strings.Cut(email, "@")
	return strings.Contains(domain, ".")
}

// Profile is what a user sees about themselves.
type Profile struct {
	Id            uuid.UUID          `json:"id"`
	Email         string             `json:"email"`
	EmailVerified bool               `json:"email_verified"`
	Name          string             `json:"name"`
	Picture       string             `json:"picture,omitempty"`
	Admin         bool               `json:"admin"`
	Identities    []IdentityProvider `json:"identities"`
	CreatedAt     time.Time          `json:"created_at"`
}

func (u *User) Profile() Profile {
	return Profile{
		Id:            u.Id,
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		Name:          u.Name,
		Picture:       u.Picture,
		Admin:         u.Admin,
		Identities:    u.Providers(),
		CreatedAt:     u.CreatedAt,
	}
}
