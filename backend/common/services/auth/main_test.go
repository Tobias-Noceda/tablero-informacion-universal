package auth

import (
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

const (
	origin   = "http://localhost"
	client   = "10.0.0.1"
	password = "correct horse battery"
)

type fixture struct {
	svc      *AuthService
	users    *mocks.MemoryUserStore
	sessions *mocks.MemorySessionStore
	tokens   *mocks.MockHandshakeStore
	mailer   *mocks.RecordingMailer
	limiter  *mocks.CountingLimiter
	verifier *staticKeys
	now      time.Time
}

// staticKeys signs by concatenation so tests can read claims back without crypto.
type staticKeys struct {
	issued []infrastructure.Claims
}

func (k *staticKeys) Sign(c infrastructure.Claims) (string, error) {
	k.issued = append(k.issued, c)
	return "access:" + c.Subject, nil
}

func newFixture(t *testing.T, admins ...string) *fixture {
	t.Helper()
	f := &fixture{
		users:    &mocks.MemoryUserStore{},
		sessions: &mocks.MemorySessionStore{},
		tokens:   &mocks.MockHandshakeStore{},
		mailer:   &mocks.RecordingMailer{},
		limiter:  &mocks.CountingLimiter{},
		verifier: &staticKeys{},
		now:      time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC),
	}
	f.svc = New(Config{AccessTTL: 15 * time.Minute, RefreshTTL: 30 * 24 * time.Hour, Admins: admins},
		f.users, f.sessions, mocks.PlainHasher{}, f.verifier, f.tokens, f.limiter, f.mailer)
	f.svc.now = func() time.Time { return f.now }
	return f
}

var linkToken = regexp.MustCompile(`token=([A-Za-z0-9_-]+)`)

func (f *fixture) lastToken(t *testing.T) string {
	t.Helper()
	mail, ok := f.mailer.Last()
	if !ok {
		t.Fatal("no mail sent")
	}
	m := linkToken.FindStringSubmatch(mail.Text)
	if m == nil {
		t.Fatalf("no token in mail: %q", mail.Text)
	}
	return m[1]
}

func (f *fixture) signUp(t *testing.T, email string) *Tokens {
	t.Helper()
	if err := f.svc.Register(email, password, "Ana", origin, client); err != nil {
		t.Fatalf("register: %v", err)
	}
	tokens, err := f.svc.VerifyEmail(f.lastToken(t))
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	return tokens
}

func TestRegister_CreatesUnverifiedUserAndMailsALink(t *testing.T) {
	f := newFixture(t)

	if err := f.svc.Register("  Ana@Example.com ", password, " Ana ", origin, client); err != nil {
		t.Fatalf("register: %v", err)
	}

	user, err := f.users.FindUserByEmail("ana@example.com")
	if err != nil {
		t.Fatalf("user not stored: %v", err)
	}
	if user.EmailVerified || user.Name != "Ana" || user.Admin {
		t.Errorf("user = %+v", user)
	}
	if identity, ok := user.Identity(models.IdentityPassword); !ok || identity.Hash == password || identity.Hash == "" {
		t.Errorf("password identity = %+v, %v", identity, ok)
	}

	mail, _ := f.mailer.Last()
	if mail.To != "ana@example.com" || !strings.Contains(mail.Text, origin+"/verify?token=") {
		t.Errorf("mail = %+v", mail)
	}
	if strings.Contains(mail.Text, password) {
		t.Error("mail must not contain the password")
	}
}

func TestRegister_ValidatesInput(t *testing.T) {
	f := newFixture(t)

	cases := []struct {
		name, email, pw, display string
		want                     error
	}{
		{"bad email", "nope", password, "Ana", ErrInvalidEmail},
		{"short password", "ana@example.com", "short", "Ana", ErrWeakPassword},
		{"long password", "ana@example.com", strings.Repeat("x", 129), "Ana", ErrWeakPassword},
		{"empty name", "ana@example.com", password, "  ", ErrInvalidName},
	}
	for _, c := range cases {
		if err := f.svc.Register(c.email, c.pw, c.display, origin, client); !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, want %v", c.name, err, c.want)
		}
	}
	if len(f.users.Users) != 0 || len(f.mailer.Mails) != 0 {
		t.Error("nothing must be stored or sent on invalid input")
	}
}

func TestRegister_DoesNotRevealExistingAccounts(t *testing.T) {
	f := newFixture(t)
	f.signUp(t, "ana@example.com")
	sent := len(f.mailer.Mails)

	if err := f.svc.Register("ana@example.com", "another password", "Eve", origin, client); err != nil {
		t.Fatalf("second register must look like success: %v", err)
	}
	if len(f.users.Users) != 1 {
		t.Error("no second user must be created")
	}
	if len(f.mailer.Mails) != sent+1 {
		t.Fatal("the existing account must be told, by mail only")
	}
	mail, _ := f.mailer.Last()
	if !strings.Contains(mail.Text, "/forgot") || strings.Contains(mail.Text, "token=") {
		t.Errorf("mail to a verified account must point at password reset, got %q", mail.Text)
	}

	if _, err := f.svc.Login("ana@example.com", "another password", client); !errors.Is(err, ErrInvalidCredentials) {
		t.Error("re-registering must not change the password of a verified account")
	}
}

func TestRegister_UnverifiedAccountCanBeReRegistered(t *testing.T) {
	f := newFixture(t)
	if err := f.svc.Register("ana@example.com", password, "Ana", origin, client); err != nil {
		t.Fatal(err)
	}
	first := f.lastToken(t)

	if err := f.svc.Register("ana@example.com", "second password!", "Ana B", origin, client); err != nil {
		t.Fatal(err)
	}
	second := f.lastToken(t)
	if first == second {
		t.Error("a fresh verification token must be issued")
	}

	if _, err := f.svc.VerifyEmail(first); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("stale token: err = %v", err)
	}
	if _, err := f.svc.VerifyEmail(second); err != nil {
		t.Fatalf("fresh token: %v", err)
	}
	if _, err := f.svc.Login("ana@example.com", "second password!", client); err != nil {
		t.Errorf("the latest password must win: %v", err)
	}
}

func TestVerifyEmail_IsSingleUseAndIssuesASession(t *testing.T) {
	f := newFixture(t)
	if err := f.svc.Register("ana@example.com", password, "Ana", origin, client); err != nil {
		t.Fatal(err)
	}
	token := f.lastToken(t)

	tokens, err := f.svc.VerifyEmail(token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if tokens.Access == "" || tokens.Refresh == "" || tokens.User.Email != "ana@example.com" || !tokens.User.EmailVerified {
		t.Errorf("tokens = %+v", tokens)
	}
	if tokens.ExpiresIn != 15*time.Minute || tokens.RefreshTTL != 30*24*time.Hour {
		t.Errorf("ttls = %v / %v", tokens.ExpiresIn, tokens.RefreshTTL)
	}
	if f.sessions.Live() != 1 {
		t.Errorf("live sessions = %d", f.sessions.Live())
	}

	if _, err := f.svc.VerifyEmail(token); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("second use: err = %v", err)
	}
	if _, err := f.svc.VerifyEmail("nope"); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("garbage: err = %v", err)
	}
}

func TestLogin_HappyPathAndFailures(t *testing.T) {
	f := newFixture(t)
	f.signUp(t, "ana@example.com")

	tokens, err := f.svc.Login("ANA@example.com", password, client)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if tokens.User.Email != "ana@example.com" || tokens.Access != "access:"+tokens.User.Id.String() {
		t.Errorf("tokens = %+v", tokens)
	}

	if _, err := f.svc.Login("ana@example.com", "wrong", client); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("wrong password: err = %v", err)
	}
	if _, err := f.svc.Login("nobody@example.com", password, client); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("unknown email: err = %v", err)
	}
}

func TestLogin_UnknownEmailStillComparesAHash(t *testing.T) {
	f := newFixture(t)
	hasher := &countingHasher{}
	f.svc.hasher = hasher

	_, _ = f.svc.Login("nobody@example.com", password, client)
	if hasher.compares != 1 {
		t.Errorf("compares = %d, want 1 (constant-time path)", hasher.compares)
	}
}

type countingHasher struct{ compares int }

func (h *countingHasher) Hash(pw string) (string, error) { return "plain:" + pw, nil }
func (h *countingHasher) Compare(encoded, pw string) (bool, error) {
	h.compares++
	return encoded == "plain:"+pw, nil
}

func TestLogin_RequiresVerifiedEmail(t *testing.T) {
	f := newFixture(t)
	if err := f.svc.Register("ana@example.com", password, "Ana", origin, client); err != nil {
		t.Fatal(err)
	}

	if _, err := f.svc.Login("ana@example.com", password, client); !errors.Is(err, ErrEmailNotVerified) {
		t.Errorf("unverified: err = %v", err)
	}
	if _, err := f.svc.Login("ana@example.com", "wrong", client); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("unverified with wrong password must not reveal the account: err = %v", err)
	}
}

func TestLogin_IsRateLimited(t *testing.T) {
	f := newFixture(t)
	f.signUp(t, "ana@example.com")
	f.limiter.Refuse = map[string]bool{"login-client:" + client: true}

	if _, err := f.svc.Login("ana@example.com", password, client); !errors.Is(err, ErrRateLimited) {
		t.Errorf("err = %v, want ErrRateLimited", err)
	}
	if f.limiter.Calls["login:ana@example.com"] != 1 || f.limiter.Calls["login-client:"+client] != 1 {
		t.Errorf("buckets = %v, want one call on the email and one on the client", f.limiter.Calls)
	}
}

func TestRefresh_RotatesAndRejectsTheOldToken(t *testing.T) {
	f := newFixture(t)
	first := f.signUp(t, "ana@example.com")

	second, err := f.svc.Refresh(first.Refresh)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if second.Refresh == first.Refresh || second.User.Id != first.User.Id {
		t.Errorf("second = %+v", second)
	}
	if f.sessions.Live() != 1 {
		t.Errorf("live sessions = %d, want 1", f.sessions.Live())
	}

	if _, err := f.svc.Refresh("garbage"); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("garbage: err = %v", err)
	}
}

func TestRefresh_ReuseWithinGraceIsHarmless(t *testing.T) {
	f := newFixture(t)
	first := f.signUp(t, "ana@example.com")
	second, _ := f.svc.Refresh(first.Refresh)

	if _, err := f.svc.Refresh(first.Refresh); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("spent token: err = %v", err)
	}
	if _, err := f.svc.Refresh(second.Refresh); err != nil {
		t.Errorf("a concurrent tab must keep its session: %v", err)
	}
}

func TestRefresh_ReuseAfterGraceRevokesTheFamily(t *testing.T) {
	f := newFixture(t)
	first := f.signUp(t, "ana@example.com")
	second, _ := f.svc.Refresh(first.Refresh)
	f.now = f.now.Add(REUSE_GRACE + time.Second)

	if _, err := f.svc.Refresh(first.Refresh); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("stolen token: err = %v", err)
	}
	if _, err := f.svc.Refresh(second.Refresh); !errors.Is(err, ErrInvalidSession) {
		t.Error("the whole family must be revoked after a late reuse")
	}
	if f.sessions.Live() != 0 {
		t.Errorf("live sessions = %d", f.sessions.Live())
	}
}

func TestRefresh_ReloadsTheUserAndAdminFlag(t *testing.T) {
	f := newFixture(t, "ana@example.com")
	tokens := f.signUp(t, "ana@example.com")
	if !tokens.User.Admin {
		t.Fatal("listed admins must be admins from the first session")
	}

	f.svc.admins = map[string]bool{}
	refreshed, err := f.svc.Refresh(tokens.Refresh)
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.User.Admin {
		t.Error("removing an email from PLATFORM_ADMINS must drop the flag on refresh")
	}
	last := f.verifier.issued[len(f.verifier.issued)-1]
	if last.Admin || last.Subject != tokens.User.Id.String() || !last.ExpiresAt.Equal(f.now.Add(15*time.Minute)) {
		t.Errorf("claims = %+v", last)
	}

	f.users.Users = nil
	if _, err := f.svc.Refresh(refreshed.Refresh); !errors.Is(err, ErrInvalidSession) {
		t.Errorf("deleted user: err = %v", err)
	}
}

func TestLogout_RevokesTheFamily(t *testing.T) {
	f := newFixture(t)
	first := f.signUp(t, "ana@example.com")
	second, _ := f.svc.Refresh(first.Refresh)

	if err := f.svc.Logout(first.Refresh); err != nil {
		t.Fatalf("logout with a spent token: %v", err)
	}
	if _, err := f.svc.Refresh(second.Refresh); !errors.Is(err, ErrInvalidSession) {
		t.Error("logout must revoke every token of the family")
	}
	if err := f.svc.Logout("garbage"); err != nil {
		t.Errorf("logout never fails: %v", err)
	}
}

func TestForgotAndReset_RevokeEverySession(t *testing.T) {
	f := newFixture(t)
	tokens := f.signUp(t, "ana@example.com")
	other, _ := f.svc.Login("ana@example.com", password, client)

	if err := f.svc.ForgotPassword("nobody@example.com", origin, client); err != nil {
		t.Errorf("unknown email must look like success: %v", err)
	}
	sent := len(f.mailer.Mails)
	if err := f.svc.ForgotPassword("Ana@example.com", origin, client); err != nil {
		t.Fatal(err)
	}
	if len(f.mailer.Mails) != sent+1 {
		t.Fatal("reset mail not sent")
	}
	mail, _ := f.mailer.Last()
	if !strings.Contains(mail.Text, origin+"/reset?token=") {
		t.Errorf("mail = %q", mail.Text)
	}
	token := f.lastToken(t)

	if _, err := f.svc.ResetPassword(token, "short"); !errors.Is(err, ErrWeakPassword) {
		t.Errorf("weak: err = %v", err)
	}
	fresh, err := f.svc.ResetPassword(token, "brand new password")
	if err != nil {
		t.Fatalf("reset: %v", err)
	}
	if _, err := f.svc.ResetPassword(token, "brand new password"); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("reused reset token: err = %v", err)
	}

	for _, old := range []*Tokens{tokens, other} {
		if _, err := f.svc.Refresh(old.Refresh); !errors.Is(err, ErrInvalidSession) {
			t.Error("old sessions must be gone after a reset")
		}
	}
	if _, err := f.svc.Refresh(fresh.Refresh); err != nil {
		t.Errorf("the session issued by the reset must work: %v", err)
	}
	if _, err := f.svc.Login("ana@example.com", "brand new password", client); err != nil {
		t.Errorf("new password: %v", err)
	}
	if _, err := f.svc.Login("ana@example.com", password, client); !errors.Is(err, ErrInvalidCredentials) {
		t.Error("old password must be gone")
	}
}

func TestReset_VerifiesTheEmailAndKeepsOtherIdentities(t *testing.T) {
	f := newFixture(t)
	user := &models.User{Id: uuid.New(), Email: "g@example.com", Name: "G", EmailVerified: true,
		Identities: []models.Identity{{Provider: models.IdentityGoogle, Subject: "sub"}}}
	_ = f.users.CreateUser(user)

	if err := f.svc.ForgotPassword("g@example.com", origin, client); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.ResetPassword(f.lastToken(t), "brand new password"); err != nil {
		t.Fatal(err)
	}

	stored, _ := f.users.FindUser(user.Id)
	if len(stored.Identities) != 2 {
		t.Errorf("identities = %+v, want google + password", stored.Identities)
	}
	if _, err := f.svc.Login("g@example.com", "brand new password", client); err != nil {
		t.Errorf("password added to a google account must work: %v", err)
	}
}
