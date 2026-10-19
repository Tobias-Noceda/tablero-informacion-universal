package users

import (
	"errors"
	"testing"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

func fixture() (*UserService, models.User, models.User) {
	ana := models.User{Id: uuid.New(), Email: "ana@example.com", Name: "Ana", EmailVerified: true, Admin: true,
		Identities: []models.Identity{{Provider: models.IdentityPassword, Hash: "h"}}}
	bob := models.User{Id: uuid.New(), Email: "bob@example.com", Name: "Bob", Picture: "https://img/bob.png"}
	store := &mocks.MemoryUserStore{Users: []models.User{ana, bob}}
	return New(store), ana, bob
}

func TestProfile_OnlyForOneself(t *testing.T) {
	svc, ana, bob := fixture()

	profile, err := svc.Profile(ana.Principal(), ana.Id)
	if err != nil {
		t.Fatalf("own profile: %v", err)
	}
	if profile.Email != ana.Email || !profile.Admin || len(profile.Identities) != 1 {
		t.Errorf("profile = %+v", profile)
	}

	if _, err := svc.Profile(ana.Principal(), bob.Id); !errors.Is(err, ErrForbidden) {
		t.Errorf("someone else's profile: err = %v", err)
	}
	if _, err := svc.Profile(models.Principal{}, ana.Id); !errors.Is(err, ErrForbidden) {
		t.Errorf("anonymous: err = %v", err)
	}
	if _, err := svc.Profile(models.Principal{ID: uuid.NewString()}, uuid.New()); !errors.Is(err, ErrForbidden) {
		t.Errorf("unknown self: err = %v", err)
	}
}

func TestSummary_ForAnyAuthenticatedUser(t *testing.T) {
	svc, ana, bob := fixture()

	summary, err := svc.Summary(ana.Principal(), bob.Id)
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if summary.Id != bob.Id || summary.Name != "Bob" || summary.Picture != bob.Picture {
		t.Errorf("summary = %+v", summary)
	}

	if _, err := svc.Summary(models.Principal{}, bob.Id); !errors.Is(err, ErrForbidden) {
		t.Errorf("anonymous: err = %v", err)
	}
	if _, err := svc.Summary(ana.Principal(), uuid.New()); !errors.Is(err, infrastructure.ErrUserNotFound) {
		t.Errorf("unknown: err = %v", err)
	}
}

func text(value string) *string { return &value }

func TestUpdate_RenamesTrimmedAndOnlyForOneself(t *testing.T) {
	svc, ana, bob := fixture()

	profile, err := svc.Update(ana.Principal(), ana.Id, Changes{Name: text("  Ana María ")})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if profile.Name != "Ana María" {
		t.Errorf("name = %q", profile.Name)
	}

	if _, err := svc.Update(ana.Principal(), ana.Id, Changes{Name: text("   ")}); !errors.Is(err, ErrInvalidName) {
		t.Errorf("blank: err = %v", err)
	}
	if _, err := svc.Update(ana.Principal(), bob.Id, Changes{Name: text("Eve")}); !errors.Is(err, ErrForbidden) {
		t.Errorf("someone else: err = %v", err)
	}
}

func TestUpdate_ChangesTheLanguage(t *testing.T) {
	svc, ana, _ := fixture()

	profile, err := svc.Update(ana.Principal(), ana.Id, Changes{Locale: text("es-AR")})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Locale != models.LocaleSpanish || profile.Name != "Ana" {
		t.Errorf("profile = %+v", profile)
	}
	if stored, _ := svc.Profile(ana.Principal(), ana.Id); stored.Locale != models.LocaleSpanish {
		t.Errorf("stored locale = %q", stored.Locale)
	}
}

// An explicit choice is not guessed at: a language the platform cannot write
// in is refused, and nothing else in the request is applied.
func TestUpdate_RefusesAnUnsupportedLanguage(t *testing.T) {
	svc, ana, _ := fixture()

	_, err := svc.Update(ana.Principal(), ana.Id, Changes{Name: text("Eve"), Locale: text("fr")})
	if !errors.Is(err, ErrInvalidLocale) {
		t.Errorf("err = %v", err)
	}
	if stored, _ := svc.Profile(ana.Principal(), ana.Id); stored.Name != "Ana" {
		t.Errorf("the name changed anyway: %q", stored.Name)
	}
}

func TestUpdate_NeedsSomethingToChange(t *testing.T) {
	svc, ana, _ := fixture()

	if _, err := svc.Update(ana.Principal(), ana.Id, Changes{}); !errors.Is(err, ErrNothingToUpdate) {
		t.Errorf("err = %v", err)
	}
}
