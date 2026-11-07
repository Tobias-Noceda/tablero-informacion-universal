//go:build integration

package mongo

import (
	"errors"
	"testing"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

func TestMongo_PostitTemplates(t *testing.T) {
	db := integrationDB(t)

	owner := "it-" + uuid.NewString()
	board, err := db.CreateBoard("templates", owner, nil)
	if err != nil {
		t.Fatal(err)
	}

	user := models.PostitTemplateScope{Kind: models.TemplateScopeUser, Owner: owner}
	boardScope := models.PostitTemplateScope{Kind: models.TemplateScopeBoard, Owner: board.Id.String()}

	put := func(name string, scope models.PostitTemplateScope) models.PostitTemplate {
		t.Helper()
		c := models.PostitTemplate{
			Key: models.NewPostitTemplateKey(), Scope: scope, Name: name,
			Resource: "https://example.com", Query: map[string]string{"a": ".a"},
			Outputs: []models.PostitTemplateOutput{{Key: "a", Label: "A", Example: 2.5}},
		}
		if err := db.PutPostitTemplate(&c); err != nil {
			t.Fatal(err)
		}
		return c
	}
	b := put("b", user)
	a := put("a", user)
	onBoard := put("board", boardScope)

	found, err := db.FindPostitTemplate(a.Key)
	if err != nil || found.Name != "a" || found.Outputs[0].Example != 2.5 || found.Scope != user {
		t.Fatalf("find = %+v, %v", found, err)
	}

	list, err := db.ListPostitTemplates(user, boardScope)
	if err != nil || len(list) != 3 || list[0].Key != a.Key || list[1].Key != b.Key {
		t.Fatalf("list = %+v, %v", list, err)
	}
	if n, _ := db.CountPostitTemplates(user); n != 2 {
		t.Errorf("count = %d", n)
	}
	if found, _ := db.FindPostitTemplates([]string{onBoard.Key, "custom_missing"}); len(found) != 1 {
		t.Errorf("find many = %+v", found)
	}

	if err := db.DeletePostitTemplate(b.Key); err != nil {
		t.Fatal(err)
	}
	if err := db.DeletePostitTemplate(b.Key); !errors.Is(err, infrastructure.ErrPostitTemplateNotFound) {
		t.Errorf("second delete: %v", err)
	}

	// Deleting the board takes its types with it, not its owner's.
	if err := db.DeleteBoard(board.Id); err != nil {
		t.Fatal(err)
	}
	if _, err := db.FindPostitTemplate(onBoard.Key); !errors.Is(err, infrastructure.ErrPostitTemplateNotFound) {
		t.Errorf("board type survived: %v", err)
	}
	if _, err := db.FindPostitTemplate(a.Key); err != nil {
		t.Errorf("user type purged: %v", err)
	}
}
