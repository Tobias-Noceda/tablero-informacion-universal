package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

func googleIdentity(subject, email string) *infrastructure.ExternalIdentity {
	return &infrastructure.ExternalIdentity{
		Provider:      models.IdentityGoogle,
		Subject:       subject,
		Email:         email,
		EmailVerified: true,
		Name:          "Ana from Google",
		Picture:       "https://lh3.example/ana.png",
	}
}

func stateOf(t *testing.T, start *GoogleSignIn) url.Values {
	t.Helper()
	parsed, err := url.Parse(start.URL)
	if err != nil {
		t.Fatalf("authorization url: %v", err)
	}
	return parsed.Query()
}

func (f *fixture) googleSignIn(t *testing.T, next string) (*Tokens, string, error) {
	t.Helper()
	start, err := f.svc.GoogleStart(origin, next)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	return f.svc.GoogleCallback(stateOf(t, start).Get("state"), "the-code", start.Binding)
}

func (f *fixture) mustGoogleSignIn(t *testing.T) *Tokens {
	t.Helper()
	tokens, _, err := f.googleSignIn(t, "")
	if err != nil {
		t.Fatalf("google sign-in: %v", err)
	}
	return tokens
}

func TestGoogleStart_BindsTheHandshakeToTheBrowserAndThePKCEVerifier(t *testing.T) {
	f := newFixture(t)
	f.google.Identity = googleIdentity("sub-1", "ana@example.com")

	start, err := f.svc.GoogleStart(origin, "/board/42")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	query := stateOf(t, start)
	state, nonce, challenge := query.Get("state"), query.Get("nonce"), query.Get("code_challenge")

	if state == "" || nonce == "" || challenge == "" || state == nonce {
		t.Fatalf("query = %v", query)
	}
	if query.Get("redirect_uri") != origin+GOOGLE_CALLBACK_PATH {
		t.Errorf("redirect_uri = %q", query.Get("redirect_uri"))
	}
	if start.Binding != hashToken(state) {
		t.Error("the browser binding must be the hash of the state")
	}

	_, next, err := f.svc.GoogleCallback(state, "the-code", start.Binding)
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	if next != "/board/42" {
		t.Errorf("next = %q", next)
	}

	seen := f.google.Seen
	sum := sha256.Sum256([]byte(seen["verifier"]))
	if base64.RawURLEncoding.EncodeToString(sum[:]) != challenge {
		t.Error("the verifier sent to the token endpoint does not match the challenge")
	}
	if seen["nonce"] != nonce || seen["code"] != "the-code" || seen["redirect_uri"] != origin+GOOGLE_CALLBACK_PATH {
		t.Errorf("complete got %v", seen)
	}
}

func TestGoogleStart_OnlyAcceptsARelativeNext(t *testing.T) {
	f := newFixture(t)

	for _, next := range []string{"https://evil.example", "//evil.example", "/\\evil.example", "evil", "/\t/evil.example", "javascript:alert(1)"} {
		if _, err := f.svc.GoogleStart(origin, next); !errors.Is(err, ErrInvalidNext) {
			t.Errorf("next %q: err = %v, want ErrInvalidNext", next, err)
		}
	}

	f.google.Identity = googleIdentity("sub-1", "ana@example.com")
	if _, next, err := f.googleSignIn(t, ""); err != nil || next != "/" {
		t.Errorf("empty next: %q, %v", next, err)
	}
}

func TestGoogle_DisabledWithoutAProvider(t *testing.T) {
	f := newFixture(t)
	f.svc.google = nil

	if _, err := f.svc.GoogleStart(origin, "/"); !errors.Is(err, ErrGoogleDisabled) {
		t.Errorf("start: err = %v", err)
	}
	if _, _, err := f.svc.GoogleCallback("state", "code", hashToken("state")); !errors.Is(err, ErrGoogleDisabled) {
		t.Errorf("callback: err = %v", err)
	}
}

func TestGoogleCallback_RequiresTheBrowserThatStartedIt(t *testing.T) {
	f := newFixture(t)
	f.google.Identity = googleIdentity("sub-1", "ana@example.com")

	start, _ := f.svc.GoogleStart(origin, "/")
	state := stateOf(t, start).Get("state")

	for _, binding := range []string{"", hashToken("another-state"), state} {
		if _, _, err := f.svc.GoogleCallback(state, "the-code", binding); !errors.Is(err, ErrInvalidState) {
			t.Errorf("binding %q: err = %v, want ErrInvalidState", binding, err)
		}
	}
	if f.google.Seen != nil {
		t.Error("the provider must not be called without the browser binding")
	}

	if _, _, err := f.svc.GoogleCallback(state, "the-code", start.Binding); err != nil {
		t.Errorf("a rejected binding must not spend the state: %v", err)
	}
	if _, _, err := f.svc.GoogleCallback(state, "the-code", start.Binding); !errors.Is(err, ErrInvalidState) {
		t.Errorf("replayed state: err = %v, want ErrInvalidState", err)
	}
}

func TestGoogleCallback_CreatesAVerifiedAccount(t *testing.T) {
	f := newFixture(t)
	f.google.Identity = googleIdentity("sub-1", " Ana@Example.com ")

	tokens := f.mustGoogleSignIn(t)

	user, err := f.users.FindUserByEmail("ana@example.com")
	if err != nil {
		t.Fatalf("user not created: %v", err)
	}
	if !user.EmailVerified || user.Name != "Ana from Google" || user.Picture != "https://lh3.example/ana.png" || user.Admin {
		t.Errorf("user = %+v", user)
	}
	if identity, ok := user.Identity(models.IdentityGoogle); !ok || identity.Subject != "sub-1" || len(user.Identities) != 1 {
		t.Errorf("identities = %+v", user.Identities)
	}
	if tokens.User.Id != user.Id || tokens.Refresh == "" || f.verifier.issued[0].Subject != user.Id.String() {
		t.Errorf("session not issued for the new user: %+v", tokens)
	}
	if _, err := f.svc.Refresh(tokens.Refresh); err != nil {
		t.Errorf("refresh: %v", err)
	}
}

func TestGoogleCallback_NamesTheAccountAfterTheMailboxWhenGoogleSendsNoName(t *testing.T) {
	f := newFixture(t)
	identity := googleIdentity("sub-1", "ana.perez@example.com")
	identity.Name = ""
	f.google.Identity = identity

	if tokens := f.mustGoogleSignIn(t); tokens.User.Name != "ana.perez" {
		t.Errorf("name = %q", tokens.User.Name)
	}
}

func TestGoogleCallback_SignsInByGoogleSubjectEvenAfterAnEmailChange(t *testing.T) {
	f := newFixture(t)
	f.google.Identity = googleIdentity("sub-1", "ana@example.com")
	first := f.mustGoogleSignIn(t)

	f.google.Identity = googleIdentity("sub-1", "ana@new-domain.example")
	second := f.mustGoogleSignIn(t)

	if second.User.Id != first.User.Id || len(f.users.Users) != 1 {
		t.Errorf("a known subject must sign into the same account: %v vs %v (%d users)", first.User.Id, second.User.Id, len(f.users.Users))
	}
}

func TestGoogleCallback_LinksAVerifiedPasswordAccount(t *testing.T) {
	f := newFixture(t)
	existing := f.signUp(t, "ana@example.com")
	f.google.Identity = googleIdentity("sub-1", "ana@example.com")

	tokens := f.mustGoogleSignIn(t)

	if tokens.User.Id != existing.User.Id {
		t.Fatal("the google identity must join the existing account")
	}
	stored, _ := f.users.FindUser(existing.User.Id)
	if len(stored.Identities) != 2 || stored.Name != "Ana" || stored.Picture != "https://lh3.example/ana.png" {
		t.Errorf("stored = %+v", stored)
	}
	if _, err := f.svc.Refresh(existing.Refresh); err != nil {
		t.Errorf("linking must not log the existing sessions out: %v", err)
	}
	if _, err := f.svc.Login("ana@example.com", password, client); err != nil {
		t.Errorf("the password must keep working: %v", err)
	}
}

func TestGoogleCallback_ReplacesAStaleGoogleIdentity(t *testing.T) {
	f := newFixture(t)
	user := &models.User{Id: uuid.New(), Email: "ana@example.com", Name: "Ana", EmailVerified: true, Picture: "https://old.example/me.png",
		Identities: []models.Identity{{Provider: models.IdentityGoogle, Subject: "deleted-google-account"}}}
	_ = f.users.CreateUser(user)
	f.google.Identity = googleIdentity("sub-2", "ana@example.com")

	f.mustGoogleSignIn(t)

	stored, _ := f.users.FindUser(user.Id)
	if identity, _ := stored.Identity(models.IdentityGoogle); len(stored.Identities) != 1 || identity.Subject != "sub-2" {
		t.Errorf("identities = %+v", stored.Identities)
	}
	if stored.Picture != "https://old.example/me.png" {
		t.Error("a picture the user already has must be kept")
	}
}

// A password registration nobody confirmed proves nothing: whoever typed the
// address may not own it. Google does vouch for the mailbox, so it wins.
func TestGoogleCallback_TakesOverAnUnverifiedRegistration(t *testing.T) {
	f := newFixture(t)
	if err := f.svc.Register("ana@example.com", "squatter password", "Squatter", origin, client); err != nil {
		t.Fatal(err)
	}
	squatted, _ := f.users.FindUserByEmail("ana@example.com")
	_ = f.sessions.PutSession(infrastructure.Session{TokenHash: "stray", Family: "stray", UserID: squatted.Id.String(), IssuedAt: f.now}, time.Hour)
	f.google.Identity = googleIdentity("sub-1", "ana@example.com")

	tokens := f.mustGoogleSignIn(t)

	stored, _ := f.users.FindUser(squatted.Id)
	if tokens.User.Id != squatted.Id || !stored.EmailVerified || stored.Name != "Ana from Google" {
		t.Errorf("stored = %+v", stored)
	}
	if _, ok := stored.Identity(models.IdentityPassword); ok || len(stored.Identities) != 1 {
		t.Errorf("the squatter's password must be dropped: %+v", stored.Identities)
	}
	if f.sessions.Live() != 1 {
		t.Errorf("only the new session may survive, %d live", f.sessions.Live())
	}
	if _, err := f.svc.Login("ana@example.com", "squatter password", client); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("squatter login: err = %v", err)
	}
}

func TestGoogleCallback_RejectsAnEmailGoogleDidNotVerify(t *testing.T) {
	f := newFixture(t)
	f.signUp(t, "ana@example.com")
	identity := googleIdentity("sub-1", "ana@example.com")
	identity.EmailVerified = false
	f.google.Identity = identity

	if _, _, err := f.googleSignIn(t, "/"); !errors.Is(err, ErrEmailNotVerified) {
		t.Fatalf("err = %v, want ErrEmailNotVerified", err)
	}
	stored, _ := f.users.FindUserByEmail("ana@example.com")
	if _, linked := stored.Identity(models.IdentityGoogle); linked {
		t.Error("an unverified google email must not be linked")
	}
}

func TestGoogleCallback_ProviderFailureCreatesNothing(t *testing.T) {
	f := newFixture(t)
	f.google.Err = errors.Join(infrastructure.ErrIdentityProvider, errors.New("invalid_grant"))

	if _, _, err := f.googleSignIn(t, "/"); !errors.Is(err, infrastructure.ErrIdentityProvider) {
		t.Errorf("err = %v", err)
	}
	if len(f.users.Users) != 0 || f.sessions.Live() != 0 {
		t.Error("a failed sign-in must leave no trace")
	}
}

func TestGoogleCallback_SeedsPlatformAdmins(t *testing.T) {
	f := newFixture(t, "ana@example.com")
	f.google.Identity = googleIdentity("sub-1", "Ana@Example.com")

	tokens := f.mustGoogleSignIn(t)

	if !tokens.User.Admin || !f.verifier.issued[0].Admin {
		t.Error("an admin email signing in with google must be an admin")
	}
}
