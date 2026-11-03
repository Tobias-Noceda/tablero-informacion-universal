//go:build integration

package mongo

import (
	"errors"
	"testing"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

func TestMongo_WellKnownsSeedAndTombstones(t *testing.T) {
	db := integrationDB(t)

	key := "it_" + uuid.NewString()[:8]
	def := "x"
	seed := models.WellKnown{
		Key: key, NameEn: "Seeded", NameEs: "Sembrado", Builtin: true, Order: 5,
		Title:      models.BoardPostItTitle{Text: "{{text}}", Vars: true},
		Parameters: []models.WellKnownParameter{{Key: "text", LabelEn: "Text", LabelEs: "Texto", Type: models.ParameterString, Default: &def}},
		Outputs:    []models.WellKnownOutput{{Key: "text", LabelEn: "Text", LabelEs: "Texto", ExampleEn: "a", ExampleEs: 1.5}},
	}

	for range 2 {
		if err := db.SeedWellKnowns([]models.WellKnown{seed}); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	found, err := db.FindWellKnown(key)
	if err != nil || found.NameEn != "Seeded" || *found.Parameters[0].Default != "x" || found.Outputs[0].ExampleEs != 1.5 {
		t.Fatalf("found = %+v, %v", found, err)
	}

	edited := *found
	edited.NameEn = "Edited"
	if err := db.PutWellKnown(&edited); err != nil {
		t.Fatalf("put: %v", err)
	}
	if err := db.SeedWellKnowns([]models.WellKnown{seed}); err != nil {
		t.Fatal(err)
	}
	if found, _ := db.FindWellKnown(key); found.NameEn != "Edited" {
		t.Errorf("seed overwrote an edit: %q", found.NameEn)
	}

	if err := db.RestoreWellKnown(key); !errors.Is(err, infrastructure.ErrWellKnownNotDeleted) {
		t.Errorf("restore live: %v", err)
	}
	if err := db.DeleteWellKnown(key); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := db.DeleteWellKnown(key); !errors.Is(err, infrastructure.ErrWellKnownNotFound) {
		t.Errorf("second delete: %v", err)
	}
	if err := db.SeedWellKnowns([]models.WellKnown{seed}); err != nil {
		t.Fatal(err)
	}

	listed := func() bool {
		wks, err := db.ListWellKnowns()
		if err != nil {
			t.Fatal(err)
		}
		for _, wk := range wks {
			if wk.Key == key {
				return true
			}
		}
		return false
	}
	if listed() {
		t.Error("a tombstone was listed (or the seed revived it)")
	}
	if found, err := db.FindWellKnown(key); err != nil || !found.Deleted {
		t.Errorf("find tombstone = %+v, %v", found, err)
	}

	if err := db.RestoreWellKnown(key); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if !listed() {
		t.Error("restored definition not listed")
	}

	if err := db.DeleteWellKnown(key); err != nil {
		t.Fatal(err)
	}
	if err := db.PutWellKnown(&edited); err != nil {
		t.Fatal(err)
	}
	if !listed() {
		t.Error("put did not revive the tombstone")
	}

	if err := db.RestoreWellKnown("it_missing_" + key); !errors.Is(err, infrastructure.ErrWellKnownNotFound) {
		t.Errorf("restore missing: %v", err)
	}
}
