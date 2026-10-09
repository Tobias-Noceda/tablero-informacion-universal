package models

import (
	"testing"

	"github.com/google/uuid"
)

func TestNormalizeEmail(t *testing.T) {
	cases := []struct{ in, want string }{
		{"  Ana@Example.COM ", "ana@example.com"},
		{"ana@example.com", "ana@example.com"},
	}
	for _, c := range cases {
		if got := NormalizeEmail(c.in); got != c.want {
			t.Errorf("NormalizeEmail(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestValidEmail(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"ana@example.com", true},
		{"ana+tag@sub.example.co", true},
		{"ana", false},
		{"ana@", false},
		{"@example.com", false},
		{"ana@example", false},
		{"", false},
	}
	for _, c := range cases {
		if got := ValidEmail(c.in); got != c.want {
			t.Errorf("ValidEmail(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestUserSummaryHidesIdentitiesAndFlags(t *testing.T) {
	user := User{
		Id:         uuid.New(),
		Email:      "ana@example.com",
		Name:       "Ana",
		Picture:    "https://img/ana.png",
		Admin:      true,
		Identities: []Identity{{Provider: IdentityPassword, Hash: "$argon2id$..."}},
	}

	summary := user.Summary()
	if summary.Id != user.Id || summary.Email != user.Email || summary.Name != user.Name || summary.Picture != user.Picture {
		t.Errorf("summary = %+v, want the public fields of %+v", summary, user)
	}
}

func TestUserProviders(t *testing.T) {
	user := User{Identities: []Identity{
		{Provider: IdentityGoogle, Subject: "123"},
		{Provider: IdentityPassword, Hash: "h"},
	}}

	got := user.Providers()
	if len(got) != 2 || got[0] != IdentityGoogle || got[1] != IdentityPassword {
		t.Errorf("Providers() = %v", got)
	}
}

func TestUserIdentity(t *testing.T) {
	user := User{Identities: []Identity{{Provider: IdentityPassword, Hash: "h"}}}

	if id, ok := user.Identity(IdentityPassword); !ok || id.Hash != "h" {
		t.Errorf("Identity(password) = %+v, %v", id, ok)
	}
	if _, ok := user.Identity(IdentityGoogle); ok {
		t.Error("Identity(google) should be absent")
	}
}

func TestPrincipalOf(t *testing.T) {
	user := User{Id: uuid.New(), Admin: true}

	principal := user.Principal()
	if principal.ID != user.Id.String() || !principal.Admin {
		t.Errorf("Principal() = %+v", principal)
	}
}

func TestUserProfileListsProviders(t *testing.T) {
	user := User{Id: uuid.New(), Email: "ana@example.com", Identities: []Identity{{Provider: IdentityGoogle, Subject: "1"}}}

	profile := user.Profile()
	if profile.Id != user.Id || len(profile.Identities) != 1 || profile.Identities[0] != IdentityGoogle {
		t.Errorf("profile = %+v", profile)
	}
}
