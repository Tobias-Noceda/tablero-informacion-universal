package wellknowns

import (
	"errors"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
)

var (
	user  = models.Principal{ID: "user"}
	admin = models.Principal{ID: "root", Admin: true}
)

func newService() (*Service, *mocks.MemoryWellKnownStore) {
	store := mocks.NewMemoryWellKnownStore()
	srv := New(store)
	if err := srv.Seed(); err != nil {
		panic(err)
	}
	return srv, store
}

func TestBuiltinsAreValid(t *testing.T) {
	seen := map[string]bool{}
	for _, wk := range Builtin() {
		if seen[wk.Key] {
			t.Errorf("%s is defined twice", wk.Key)
		}
		seen[wk.Key] = true
		if err := wk.Validate(); err != nil {
			t.Errorf("%s: %v", wk.Key, err)
		}
		if !wk.Builtin {
			t.Errorf("%s is not marked builtin", wk.Key)
		}
	}
}

// builtin is one of the seeded definitions, by key.
func builtin(t *testing.T, key string) models.WellKnown {
	t.Helper()
	for _, wk := range Builtin() {
		if wk.Key == key {
			return wk
		}
	}
	t.Fatalf("no built-in %q", key)
	return models.WellKnown{}
}

// The image card answers with the address it is given, and needs one.
func TestBuiltinImage_AnswersWithItsAddress(t *testing.T) {
	image := builtin(t, "image")
	if image.Resource != "" {
		t.Errorf("resource = %q, want none: the card makes no request", image.Resource)
	}
	if _, _, err := image.Instantiate(nil); !errors.Is(err, models.ErrWellKnownMissingVariable) {
		t.Errorf("without an address: err = %v, want ErrWellKnownMissingVariable", err)
	}

	postit, _, err := image.Instantiate(map[string]string{"url": "https://example.com/cat.png"})
	if err != nil {
		t.Fatal(err)
	}
	if postit.Resource != nil || postit.Params["url"] != "https://example.com/cat.png" {
		t.Errorf("postit = %+v, want no resource and the address in its params", postit)
	}
}

func TestList_HidesSystemSecretsFromUsers(t *testing.T) {
	srv, _ := newService()

	for _, c := range []struct {
		who  models.Principal
		want bool
	}{{user, false}, {admin, true}} {
		wks, err := srv.List(c.who)
		if err != nil {
			t.Fatal(err)
		}
		if len(wks) != len(Builtin()) {
			t.Fatalf("listed %d, want %d", len(wks), len(Builtin()))
		}
		for _, wk := range wks {
			if wk.Key == "nasa_apod" && (len(wk.SystemSecrets) > 0) != c.want {
				t.Errorf("admin=%v sees system secrets = %v", c.who.Admin, wk.SystemSecrets)
			}
		}
	}
}

func TestWrites_AreAdminOnly(t *testing.T) {
	srv, _ := newService()
	wk := Builtin()[0]

	if _, err := srv.Put(user, "x", wk); !errors.Is(err, ErrForbidden) {
		t.Errorf("put: err = %v", err)
	}
	if err := srv.Delete(user, wk.Key); !errors.Is(err, ErrForbidden) {
		t.Errorf("delete: err = %v", err)
	}
	if _, err := srv.Restore(user, wk.Key); !errors.Is(err, ErrForbidden) {
		t.Errorf("restore: err = %v", err)
	}
}

func TestPut_ValidatesAndStamps(t *testing.T) {
	srv, store := newService()

	bad := builtin(t, "temperature")
	bad.Query = map[string]string{}
	if _, err := srv.Put(admin, "broken", bad); !errors.Is(err, ErrInvalid) {
		t.Errorf("err = %v, want invalid", err)
	}

	edited := builtin(t, "temperature")
	edited.Builtin = false
	edited.NameEn = "Renamed"
	got, err := srv.Put(admin, edited.Key, edited)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Builtin || got.UpdatedBy != "root" || got.UpdatedAt.IsZero() {
		t.Errorf("stored = %+v, want builtin kept and the edit stamped", got)
	}

	fresh := builtin(t, "temperature")
	fresh.Builtin = true
	got, err = srv.Put(admin, "my_card", fresh)
	if err != nil {
		t.Fatal(err)
	}
	if got.Key != "my_card" || got.Builtin {
		t.Errorf("stored = %+v, want the path key and not builtin", got)
	}

	// The seed leaves an edited definition alone.
	if err := srv.Seed(); err != nil {
		t.Fatal(err)
	}
	if stored, _ := store.FindWellKnown(edited.Key); stored.NameEn != "Renamed" {
		t.Errorf("seed overwrote the edit: %q", stored.NameEn)
	}
}

func TestDelete_TombstoneSurvivesSeedAndRestores(t *testing.T) {
	srv, store := newService()

	if err := srv.Delete(admin, "dog_facts"); err != nil {
		t.Fatal(err)
	}
	if err := srv.Delete(admin, "dog_facts"); !errors.Is(err, ErrNotFound) {
		t.Errorf("second delete: err = %v", err)
	}
	if err := srv.Seed(); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.Get(user, "dog_facts"); !errors.Is(err, ErrNotFound) {
		t.Errorf("get after delete + seed: err = %v", err)
	}
	if wks, _ := srv.List(user); len(wks) != len(Builtin())-1 {
		t.Errorf("listed %d after a delete", len(wks))
	}
	// Running cards still read it.
	if stored, err := store.FindWellKnown("dog_facts"); err != nil || !stored.Deleted {
		t.Errorf("store = %+v, %v; want the tombstone kept", stored, err)
	}

	if _, err := srv.Restore(admin, "dog_facts"); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if _, err := srv.Restore(admin, "dog_facts"); !errors.Is(err, ErrNotDeleted) {
		t.Errorf("restore live: err = %v", err)
	}
	if _, err := srv.Restore(admin, "nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("restore missing: err = %v", err)
	}
	if _, err := srv.Get(user, "dog_facts"); err != nil {
		t.Errorf("get after restore: %v", err)
	}
}
