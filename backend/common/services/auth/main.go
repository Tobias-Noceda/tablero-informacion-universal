package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

var (
	ErrInvalidEmail       = errors.New("invalid email")
	ErrInvalidName        = errors.New("invalid name")
	ErrWeakPassword       = errors.New("password must have between 8 and 128 characters")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailNotVerified   = errors.New("email not verified")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrInvalidSession     = errors.New("invalid session")
	ErrRateLimited        = infrastructure.ErrRateLimited
)

const (
	VERIFY_TTL  = 24 * time.Hour
	RESET_TTL   = time.Hour
	REUSE_GRACE = 10 * time.Second

	loginLimit    = 10
	loginWindow   = time.Minute
	mailLimit     = 5
	mailWindow    = time.Hour
	minPassword   = 8
	maxPassword   = 128
	tokenBytes    = 32
	dummyPassword = "never-matches"
)

type Config struct {
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	Admins     []string
}

// Tokens is what a successful sign-in hands back: the access token for the
// Authorization header and the refresh token the controller turns into a cookie.
type Tokens struct {
	Access     string
	ExpiresIn  time.Duration
	Refresh    string
	RefreshTTL time.Duration
	User       *models.User
}

type AuthService struct {
	config   Config
	admins   map[string]bool
	users    infrastructure.UserStore
	sessions infrastructure.SessionStore
	hasher   infrastructure.PasswordHasher
	signer   infrastructure.TokenSigner
	tokens   infrastructure.HandshakeStore
	limiter  infrastructure.RateLimiter
	mailer   infrastructure.Mailer
	dummy    string
	now      func() time.Time
}

func New(config Config, users infrastructure.UserStore, sessions infrastructure.SessionStore, hasher infrastructure.PasswordHasher,
	signer infrastructure.TokenSigner, tokens infrastructure.HandshakeStore, limiter infrastructure.RateLimiter, mailer infrastructure.Mailer) *AuthService {
	admins := make(map[string]bool, len(config.Admins))
	for _, email := range config.Admins {
		if normalized := models.NormalizeEmail(email); normalized != "" {
			admins[normalized] = true
		}
	}

	dummy, _ := hasher.Hash(dummyPassword)

	return &AuthService{
		config:   config,
		admins:   admins,
		users:    users,
		sessions: sessions,
		hasher:   hasher,
		signer:   signer,
		tokens:   tokens,
		limiter:  limiter,
		mailer:   mailer,
		dummy:    dummy,
		now:      time.Now,
	}
}

// Register never says whether the email was known: the answer goes by mail.
func (srv *AuthService) Register(email, password, name, origin, client string) error {
	email = models.NormalizeEmail(email)
	name = strings.TrimSpace(name)
	if !models.ValidEmail(email) {
		return ErrInvalidEmail
	}
	if !validPassword(password) {
		return ErrWeakPassword
	}
	if name == "" {
		return ErrInvalidName
	}
	if err := srv.limit(mailLimit, mailWindow, "register:"+email, "register-client:"+client); err != nil {
		return err
	}

	hash, err := srv.hasher.Hash(password)
	if err != nil {
		return err
	}

	existing, err := srv.users.FindUserByEmail(email)
	switch {
	case errors.Is(err, infrastructure.ErrUserNotFound):
		now := srv.now()
		user := &models.User{
			Id:         uuid.New(),
			Email:      email,
			Name:       name,
			Admin:      srv.admins[email],
			Identities: []models.Identity{{Provider: models.IdentityPassword, Hash: hash}},
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if err := srv.users.CreateUser(user); err != nil {
			return err
		}
		return srv.sendVerification(user, origin)
	case err != nil:
		return err
	case existing.EmailVerified:
		return srv.mailer.Send(models.Mail{
			To:      email,
			Subject: "You already have an account",
			Text:    "Someone tried to sign up with this address, which already has an account. If it was you and you forgot your password, reset it at " + origin + "/forgot",
		})
	default:
		if err := srv.users.ReplaceIdentities(existing.Id, withPassword(existing.Identities, hash)); err != nil {
			return err
		}
		if err := srv.users.UpdateUser(existing.Id, map[string]any{"name": name}); err != nil {
			return err
		}
		return srv.sendVerification(existing, origin)
	}
}

func (srv *AuthService) VerifyEmail(token string) (*Tokens, error) {
	userID, err := srv.redeem("verify", token)
	if err != nil {
		return nil, err
	}

	user, err := srv.users.FindUser(userID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if err := srv.users.UpdateUser(user.Id, map[string]any{"emailverified": true}); err != nil {
		return nil, err
	}
	user.EmailVerified = true

	return srv.issue(user, uuid.NewString())
}

func (srv *AuthService) Login(email, password, client string) (*Tokens, error) {
	email = models.NormalizeEmail(email)
	if err := srv.limit(loginLimit, loginWindow, "login:"+email, "login-client:"+client); err != nil {
		return nil, err
	}

	user, err := srv.users.FindUserByEmail(email)
	if err != nil && !errors.Is(err, infrastructure.ErrUserNotFound) {
		return nil, err
	}

	encoded := srv.dummy
	if user != nil {
		if identity, ok := user.Identity(models.IdentityPassword); ok {
			encoded = identity.Hash
		}
	}

	matches, err := srv.hasher.Compare(encoded, password)
	if err != nil {
		return nil, err
	}
	if user == nil || !matches || encoded == srv.dummy {
		return nil, ErrInvalidCredentials
	}
	if !user.EmailVerified {
		return nil, ErrEmailNotVerified
	}

	return srv.issue(user, uuid.NewString())
}

// Refresh rotates the token. A token that was already spent is either a
// harmless race between tabs or a stolen copy; past the grace window the
// whole family is revoked.
func (srv *AuthService) Refresh(refresh string) (*Tokens, error) {
	hash := hashToken(refresh)

	session, err := srv.sessions.TakeSession(hash)
	if errors.Is(err, infrastructure.ErrSessionNotFound) {
		family, at, spent, err := srv.sessions.Spent(hash)
		if err != nil {
			return nil, err
		}
		if spent && srv.now().Sub(at) > REUSE_GRACE {
			if err := srv.sessions.RevokeFamily(family); err != nil {
				return nil, err
			}
		}
		return nil, ErrInvalidSession
	}
	if err != nil {
		return nil, err
	}

	if err := srv.sessions.MarkSpent(hash, session.Family, srv.now(), srv.config.RefreshTTL); err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(session.UserID)
	if err != nil {
		return nil, ErrInvalidSession
	}
	user, err := srv.users.FindUser(userID)
	if errors.Is(err, infrastructure.ErrUserNotFound) {
		_ = srv.sessions.RevokeFamily(session.Family)
		return nil, ErrInvalidSession
	}
	if err != nil {
		return nil, err
	}

	return srv.issue(user, session.Family)
}

func (srv *AuthService) Logout(refresh string) error {
	hash := hashToken(refresh)

	if session, err := srv.sessions.TakeSession(hash); err == nil {
		return srv.sessions.RevokeFamily(session.Family)
	}
	if family, _, spent, _ := srv.sessions.Spent(hash); spent {
		return srv.sessions.RevokeFamily(family)
	}

	return nil
}

func (srv *AuthService) ForgotPassword(email, origin, client string) error {
	email = models.NormalizeEmail(email)
	if err := srv.limit(mailLimit, mailWindow, "forgot:"+email, "forgot-client:"+client); err != nil {
		return err
	}

	user, err := srv.users.FindUserByEmail(email)
	if errors.Is(err, infrastructure.ErrUserNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	token, err := srv.mint("reset", user.Id, RESET_TTL)
	if err != nil {
		return err
	}

	return srv.mailer.Send(models.Mail{
		To:      user.Email,
		Subject: "Reset your password",
		Text:    "Choose a new password at " + origin + "/reset?token=" + token + " (valid for one hour). If you did not ask for this, ignore this mail.",
	})
}

// ResetPassword proves ownership of the mailbox, so it also verifies the
// email, and it logs every other session out.
func (srv *AuthService) ResetPassword(token, password string) (*Tokens, error) {
	if !validPassword(password) {
		return nil, ErrWeakPassword
	}

	userID, err := srv.redeem("reset", token)
	if err != nil {
		return nil, err
	}
	user, err := srv.users.FindUser(userID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	hash, err := srv.hasher.Hash(password)
	if err != nil {
		return nil, err
	}
	if err := srv.users.ReplaceIdentities(user.Id, withPassword(user.Identities, hash)); err != nil {
		return nil, err
	}
	if err := srv.users.UpdateUser(user.Id, map[string]any{"emailverified": true}); err != nil {
		return nil, err
	}
	if err := srv.sessions.RevokeUser(user.Id.String()); err != nil {
		return nil, err
	}
	user.EmailVerified = true

	return srv.issue(user, uuid.NewString())
}

func (srv *AuthService) issue(user *models.User, family string) (*Tokens, error) {
	if err := srv.syncAdmin(user); err != nil {
		return nil, err
	}

	now := srv.now()
	access, err := srv.signer.Sign(infrastructure.Claims{
		Subject:   user.Id.String(),
		Admin:     user.Admin,
		IssuedAt:  now,
		ExpiresAt: now.Add(srv.config.AccessTTL),
		ID:        uuid.NewString(),
	})
	if err != nil {
		return nil, err
	}

	refresh, err := randomToken()
	if err != nil {
		return nil, err
	}
	session := infrastructure.Session{TokenHash: hashToken(refresh), Family: family, UserID: user.Id.String(), IssuedAt: now}
	if err := srv.sessions.PutSession(session, srv.config.RefreshTTL); err != nil {
		return nil, err
	}

	return &Tokens{
		Access:     access,
		ExpiresIn:  srv.config.AccessTTL,
		Refresh:    refresh,
		RefreshTTL: srv.config.RefreshTTL,
		User:       user,
	}, nil
}

func (srv *AuthService) syncAdmin(user *models.User) error {
	admin := srv.admins[user.Email]
	if admin == user.Admin {
		return nil
	}
	if err := srv.users.UpdateUser(user.Id, map[string]any{"admin": admin}); err != nil {
		return err
	}
	user.Admin = admin
	return nil
}

func (srv *AuthService) sendVerification(user *models.User, origin string) error {
	token, err := srv.mint("verify", user.Id, VERIFY_TTL)
	if err != nil {
		return err
	}

	return srv.mailer.Send(models.Mail{
		To:      user.Email,
		Subject: "Verify your email",
		Text:    "Welcome, " + user.Name + ". Confirm your address at " + origin + "/verify?token=" + token + " (valid for 24 hours).",
	})
}

// mint issues a single-use token for a user and forgets the previous one of
// the same kind, so only the latest mail works.
func (srv *AuthService) mint(kind string, userID uuid.UUID, ttl time.Duration) (string, error) {
	if previous, err := srv.tokens.Take(currentKey(kind, userID)); err == nil {
		_, _ = srv.tokens.Take(tokenKey(kind, string(previous)))
	}

	token, err := randomToken()
	if err != nil {
		return "", err
	}
	hash := hashToken(token)

	if err := srv.tokens.Put(tokenKey(kind, hash), []byte(userID.String()), ttl); err != nil {
		return "", err
	}
	if err := srv.tokens.Put(currentKey(kind, userID), []byte(hash), ttl); err != nil {
		return "", err
	}

	return token, nil
}

func (srv *AuthService) redeem(kind, token string) (uuid.UUID, error) {
	raw, err := srv.tokens.Take(tokenKey(kind, hashToken(token)))
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}
	userID, err := uuid.Parse(string(raw))
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}
	_, _ = srv.tokens.Take(currentKey(kind, userID))

	return userID, nil
}

func (srv *AuthService) limit(limit int, window time.Duration, keys ...string) error {
	for _, key := range keys {
		allowed, err := srv.limiter.Allow(key, limit, window)
		if err != nil {
			return err
		}
		if !allowed {
			return ErrRateLimited
		}
	}
	return nil
}

func withPassword(identities []models.Identity, hash string) []models.Identity {
	out := make([]models.Identity, 0, len(identities)+1)
	for _, identity := range identities {
		if identity.Provider != models.IdentityPassword {
			out = append(out, identity)
		}
	}
	return append(out, models.Identity{Provider: models.IdentityPassword, Hash: hash})
}

func validPassword(password string) bool {
	return len(password) >= minPassword && len(password) <= maxPassword
}

func tokenKey(kind, hash string) string             { return kind + ":" + hash }
func currentKey(kind string, user uuid.UUID) string { return kind + "-current:" + user.String() }

func randomToken() (string, error) {
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
