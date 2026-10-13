//go:build integration

package mongo

import (
	"errors"
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

func newUser(email string) *models.User {
	now := time.Now().UTC().Truncate(time.Millisecond)
	return &models.User{
		Id:         uuid.New(),
		Email:      email,
		Name:       "Ana",
		Identities: []models.Identity{{Provider: models.IdentityPassword, Hash: "h"}},
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func TestMongo_UsersRoundTrip(t *testing.T) {
	db := integrationDB(t)

	user := newUser("ana@example.com")
	if err := db.CreateUser(user); err != nil {
		t.Fatalf("create: %v", err)
	}

	byID, err := db.FindUser(user.Id)
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if byID.Email != user.Email || len(byID.Identities) != 1 || byID.Identities[0].Hash != "h" {
		t.Errorf("found = %+v", byID)
	}

	byEmail, err := db.FindUserByEmail(user.Email)
	if err != nil || byEmail.Id != user.Id {
		t.Errorf("find by email = %+v, %v", byEmail, err)
	}

	if _, err := db.FindUser(uuid.New()); !errors.Is(err, infrastructure.ErrUserNotFound) {
		t.Errorf("unknown id: err = %v", err)
	}
	if _, err := db.FindUserByEmail("nobody@example.com"); !errors.Is(err, infrastructure.ErrUserNotFound) {
		t.Errorf("unknown email: err = %v", err)
	}
}

func TestMongo_UsersEmailIsUnique(t *testing.T) {
	db := integrationDB(t)

	if err := db.CreateUser(newUser("dup@example.com")); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if err := db.CreateUser(newUser("dup@example.com")); !errors.Is(err, infrastructure.ErrEmailTaken) {
		t.Errorf("second create: err = %v, want ErrEmailTaken", err)
	}
}

func TestMongo_UsersIdentityLookupAndReplace(t *testing.T) {
	db := integrationDB(t)

	user := newUser("g@example.com")
	user.Identities = []models.Identity{{Provider: models.IdentityGoogle, Subject: "sub-1"}}
	if err := db.CreateUser(user); err != nil {
		t.Fatalf("create: %v", err)
	}

	found, err := db.FindUserByIdentity(models.IdentityGoogle, "sub-1")
	if err != nil || found.Id != user.Id {
		t.Fatalf("find by identity = %+v, %v", found, err)
	}
	if _, err := db.FindUserByIdentity(models.IdentityGoogle, "sub-2"); !errors.Is(err, infrastructure.ErrUserNotFound) {
		t.Errorf("unknown subject: err = %v", err)
	}

	other := newUser("other@example.com")
	other.Identities = []models.Identity{{Provider: models.IdentityGoogle, Subject: "sub-1"}}
	if err := db.CreateUser(other); err == nil {
		t.Error("two users must not share a google subject")
	}

	replaced := []models.Identity{
		{Provider: models.IdentityGoogle, Subject: "sub-1"},
		{Provider: models.IdentityPassword, Hash: "new"},
	}
	if err := db.ReplaceIdentities(user.Id, replaced); err != nil {
		t.Fatalf("replace: %v", err)
	}
	found, _ = db.FindUser(user.Id)
	if len(found.Identities) != 2 || found.Identities[1].Hash != "new" {
		t.Errorf("identities after replace = %+v", found.Identities)
	}
}

func TestMongo_UsersUpdateAndBatchFind(t *testing.T) {
	db := integrationDB(t)

	a, b := newUser("a@example.com"), newUser("b@example.com")
	for _, u := range []*models.User{a, b} {
		if err := db.CreateUser(u); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	if err := db.UpdateUser(a.Id, map[string]any{"name": "Alice", "admin": true, "emailverified": true}); err != nil {
		t.Fatalf("update: %v", err)
	}
	found, _ := db.FindUser(a.Id)
	if found.Name != "Alice" || !found.Admin || !found.EmailVerified || !found.UpdatedAt.After(a.UpdatedAt) {
		t.Errorf("after update = %+v", found)
	}
	if err := db.UpdateUser(uuid.New(), map[string]any{"name": "x"}); !errors.Is(err, infrastructure.ErrUserNotFound) {
		t.Errorf("update unknown: err = %v", err)
	}

	users, err := db.FindUsers([]string{a.Id.String(), b.Id.String(), uuid.NewString()})
	if err != nil || len(users) != 2 {
		t.Errorf("FindUsers = %d users, %v", len(users), err)
	}
}
