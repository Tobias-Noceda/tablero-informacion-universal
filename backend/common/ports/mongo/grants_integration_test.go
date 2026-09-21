//go:build integration

package mongo

import (
	"errors"
	"testing"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

func TestMongo_GrantsSurviveARewriteAndAreFoundByAudience(t *testing.T) {
	db := integrationDB(t)
	srv := integrationService(t, db, kek(1, 0xB2))
	scope := models.UserScope("alice")
	alice := models.Principal{ID: "alice"}
	group := uuid.New().String()

	if err := srv.Put(scope, alice, "KEY", models.SecretApiKey, "one"); err != nil {
		t.Fatalf("put: %v", err)
	}

	grants := []models.Grant{
		{To: models.Audience{Kind: models.AudienceUser, ID: "bob"}},
		{To: models.Audience{Kind: models.AudienceGroup, ID: group}, Board: uuid.New().String()},
	}
	if err := db.SetGrants(scope, "KEY", grants); err != nil {
		t.Fatalf("set grants: %v", err)
	}

	if err := srv.Put(scope, alice, "KEY", models.SecretApiKey, "two"); err != nil {
		t.Fatalf("rewrite: %v", err)
	}

	stored, _ := db.FindSecrets(scope, []string{"KEY"})
	if len(stored) != 1 || len(stored[0].Grants) != 2 {
		t.Fatalf("after rewrite: %+v, want both grants kept", stored)
	}

	for _, audience := range []models.Audience{
		{Kind: models.AudienceUser, ID: "bob"},
		{Kind: models.AudienceGroup, ID: group},
	} {
		found, err := db.FindGranted([]models.Audience{audience})
		if err != nil {
			t.Fatalf("find granted: %v", err)
		}
		if len(found) != 1 || found[0].Name != "KEY" {
			t.Errorf("%+v sees %+v, want KEY", audience, found)
		}
	}
	if found, _ := db.FindGranted([]models.Audience{{Kind: models.AudienceUser, ID: "eve"}}); len(found) != 0 {
		t.Errorf("eve sees %+v, want nothing", found)
	}
	if found, _ := db.FindGranted(nil); len(found) != 0 {
		t.Errorf("no audience sees %+v, want nothing", found)
	}

	if err := db.SetGrants(scope, "KEY", nil); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if found, _ := db.FindGranted([]models.Audience{{Kind: models.AudienceUser, ID: "bob"}}); len(found) != 0 {
		t.Errorf("bob still sees %+v after revoke", found)
	}

	if err := db.SetGrants(scope, "NOPE", grants); !errors.Is(err, infrastructure.ErrUnknownCredential) {
		t.Errorf("granting a missing secret: %v, want ErrUnknownCredential", err)
	}
}
