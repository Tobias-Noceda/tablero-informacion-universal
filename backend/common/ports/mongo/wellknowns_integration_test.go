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
		TitleEn:    "{{text}}",
		TitleEs:    "{{text}}",
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

// A built-in seeded before date types existed has its date parameters as
// text; the next seed types them, and leaves alone a type an admin chose.
func TestMongo_WellKnownsSeedTypesOldDateParameters(t *testing.T) {
	db := integrationDB(t)

	key := "it_" + uuid.NewString()[:8]
	old := models.WellKnown{
		Key: key, NameEn: "Dated", NameEs: "Con fecha", Builtin: true,
		TitleEn: "{{x}}", TitleEs: "{{x}}",
		Resource: "https://example.com", Request: models.Request{Method: "GET", Queries: map[string]string{"from": "$from", "to": "$to", "at": "$at"}},
		Response: "json", Query: map[string]string{"x": ".x"},
		Parameters: []models.WellKnownParameter{
			{Key: "$from", LabelEn: "From", LabelEs: "Desde", Type: models.ParameterString},
			{Key: "$to", LabelEn: "To", LabelEs: "Hasta", Type: models.ParameterNumber},
			{Key: "$at", LabelEn: "At", LabelEs: "A las", Type: models.ParameterString},
		},
		Outputs: []models.WellKnownOutput{{Key: "x", LabelEn: "X", LabelEs: "X"}},
	}
	if err := db.SeedWellKnowns([]models.WellKnown{old}); err != nil {
		t.Fatalf("seed old: %v", err)
	}

	now := old
	now.Parameters = []models.WellKnownParameter{
		{Key: "$from", LabelEn: "From", LabelEs: "Desde", Type: models.ParameterDate},
		{Key: "$to", LabelEn: "To", LabelEs: "Hasta", Type: models.ParameterDate},
		{Key: "$at", LabelEn: "At", LabelEs: "A las", Type: models.ParameterDateTime},
	}
	for range 2 {
		if err := db.SeedWellKnowns([]models.WellKnown{now}); err != nil {
			t.Fatalf("seed new: %v", err)
		}
	}

	found, err := db.FindWellKnown(key)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]models.WellKnownParameterType{}
	for _, p := range found.Parameters {
		got[p.Key] = p.Type
	}
	want := map[string]models.WellKnownParameterType{"$from": models.ParameterDate, "$to": models.ParameterNumber, "$at": models.ParameterDateTime}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: type %q, want %q", k, got[k], v)
		}
	}
}
