package auth

import (
	"testing"

	"github.com/Secreto31126/tesis/common/models"
)

func (f *fixture) lastMail(t *testing.T) models.Mail {
	t.Helper()
	mail, ok := f.mailer.Last()
	if !ok {
		t.Fatal("no mail sent")
	}
	return mail
}

func TestRegister_StoresTheLanguageAndVerifiesInIt(t *testing.T) {
	f := newFixture(t)

	if err := f.svc.Register("ana@example.com", password, "Ana", "es-AR", origin, client); err != nil {
		t.Fatal(err)
	}

	user, _ := f.users.FindUserByEmail("ana@example.com")
	if user.Locale != models.LocaleSpanish {
		t.Errorf("stored locale = %q", user.Locale)
	}
	mail := f.lastMail(t)
	if mail.Template != models.MailVerifyEmail || mail.Locale != models.LocaleSpanish || mail.Vars["Name"] != "Ana" {
		t.Errorf("mail = %+v", mail)
	}
}

func TestRegister_AnUnsupportedLanguageIsNotAnError(t *testing.T) {
	f := newFixture(t)

	if err := f.svc.Register("ana@example.com", password, "Ana", "klingon", origin, client); err != nil {
		t.Fatal(err)
	}

	user, _ := f.users.FindUserByEmail("ana@example.com")
	if user.Locale != models.DefaultLocale {
		t.Errorf("stored locale = %q", user.Locale)
	}
}

// The mail goes to the account's owner, so it speaks their language, not
// whoever typed the address into the form.
func TestRegister_TellsAnExistingAccountInItsOwnLanguage(t *testing.T) {
	f := newFixture(t)
	if err := f.svc.Register("ana@example.com", password, "Ana", "es", origin, client); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.VerifyEmail(f.lastToken(t)); err != nil {
		t.Fatal(err)
	}

	if err := f.svc.Register("ana@example.com", "another password", "Eve", "en", origin, client); err != nil {
		t.Fatal(err)
	}

	mail := f.lastMail(t)
	if mail.Template != models.MailAccountExists || mail.Locale != models.LocaleSpanish || mail.Vars["Name"] != "Ana" {
		t.Errorf("mail = %+v", mail)
	}
	if mail.Vars["Link"] != origin+"/forgot" {
		t.Errorf("link = %q", mail.Vars["Link"])
	}
	if user, _ := f.users.FindUserByEmail("ana@example.com"); user.Locale != models.LocaleSpanish {
		t.Errorf("a verified account's language must not change, got %q", user.Locale)
	}
}

func TestRegister_AnUnverifiedRetakeAdoptsTheNewLanguage(t *testing.T) {
	f := newFixture(t)
	_ = f.svc.Register("ana@example.com", password, "Ana", "en", origin, client)

	if err := f.svc.Register("ana@example.com", password, "Ana", "es", origin, client); err != nil {
		t.Fatal(err)
	}

	if user, _ := f.users.FindUserByEmail("ana@example.com"); user.Locale != models.LocaleSpanish {
		t.Errorf("stored locale = %q", user.Locale)
	}
	if mail := f.lastMail(t); mail.Locale != models.LocaleSpanish {
		t.Errorf("mail locale = %q", mail.Locale)
	}
}

func TestForgotPassword_MailsInTheAccountsLanguage(t *testing.T) {
	f := newFixture(t)
	_ = f.svc.Register("ana@example.com", password, "Ana", "es", origin, client)

	if err := f.svc.ForgotPassword("ana@example.com", origin, client); err != nil {
		t.Fatal(err)
	}

	mail := f.lastMail(t)
	if mail.Template != models.MailResetPassword || mail.Locale != models.LocaleSpanish || mail.Vars["Name"] != "Ana" {
		t.Errorf("mail = %+v", mail)
	}
}

func TestGoogleCallback_CreatesTheAccountInTheLanguageItStartedIn(t *testing.T) {
	f := newFixture(t)
	f.google.Identity = googleIdentity("sub-1", "ana@example.com")

	start, err := f.svc.GoogleStart(origin, "/", "es")
	if err != nil {
		t.Fatal(err)
	}
	tokens, _, err := f.svc.GoogleCallback(stateOf(t, start).Get("state"), "the-code", start.Binding)
	if err != nil {
		t.Fatal(err)
	}

	if stored, _ := f.users.FindUser(tokens.User.Id); stored.Locale != models.LocaleSpanish {
		t.Errorf("stored locale = %q", stored.Locale)
	}
}

func TestGoogleCallback_KeepsTheLanguageOfAnExistingAccount(t *testing.T) {
	f := newFixture(t)
	f.google.Identity = googleIdentity("sub-1", "ana@example.com")
	first, _ := f.svc.GoogleStart(origin, "/", "es")
	tokens, _, _ := f.svc.GoogleCallback(stateOf(t, first).Get("state"), "the-code", first.Binding)

	again, _ := f.svc.GoogleStart(origin, "/", "en")
	if _, _, err := f.svc.GoogleCallback(stateOf(t, again).Get("state"), "the-code", again.Binding); err != nil {
		t.Fatal(err)
	}

	if stored, _ := f.users.FindUser(tokens.User.Id); stored.Locale != models.LocaleSpanish {
		t.Errorf("signing in must not change the stored language, got %q", stored.Locale)
	}
}
