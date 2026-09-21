package models

import (
	"testing"

	"github.com/google/uuid"
)

func TestGrantValid(t *testing.T) {
	group := uuid.New().String()
	board := uuid.New().String()

	cases := []struct {
		label string
		grant Grant
		want  bool
	}{
		{"user anywhere", Grant{To: Audience{Kind: AudienceUser, ID: "ana"}}, true},
		{"user on one board", Grant{To: Audience{Kind: AudienceUser, ID: "ana"}, Board: board}, true},
		{"group", Grant{To: Audience{Kind: AudienceGroup, ID: group}}, true},
		{"board members", Grant{To: Audience{Kind: AudienceBoard, ID: board}}, true},
		{"board members restricted to itself", Grant{To: Audience{Kind: AudienceBoard, ID: board}, Board: board}, true},
		{"board members restricted to another board", Grant{To: Audience{Kind: AudienceBoard, ID: board}, Board: uuid.New().String()}, false},
		{"empty user", Grant{To: Audience{Kind: AudienceUser}}, false},
		{"group with non uuid id", Grant{To: Audience{Kind: AudienceGroup, ID: "ops"}}, false},
		{"board with non uuid id", Grant{To: Audience{Kind: AudienceBoard, ID: "main"}}, false},
		{"non uuid board restriction", Grant{To: Audience{Kind: AudienceUser, ID: "ana"}, Board: "main"}, false},
		{"unknown kind", Grant{To: Audience{Kind: "org", ID: group}}, false},
	}

	for _, c := range cases {
		if got := c.grant.Valid(); got != c.want {
			t.Errorf("%s: Valid() = %v, want %v", c.label, got, c.want)
		}
	}
}

func TestSecretMeta_CarriesGrants(t *testing.T) {
	grant := Grant{To: Audience{Kind: AudienceUser, ID: "ana"}}
	secret := Secret{Name: "KEY", Grants: []Grant{grant}}

	meta := secret.Meta()
	if len(meta.Grants) != 1 || meta.Grants[0] != grant {
		t.Errorf("meta.Grants = %v, want %v", meta.Grants, grant)
	}
}
