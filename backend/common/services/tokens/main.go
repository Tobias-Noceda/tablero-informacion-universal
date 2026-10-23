package tokens

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

const (
	// PREFIX starts every personal access token, so the middleware can tell
	// one from a session JWT without trying to parse it.
	PREFIX = "tiu_pat_"

	DEFAULT_DAYS = 15
	MAX_TOKENS   = 10
	MAX_NAME     = 64

	createLimit  = 10
	createWindow = time.Hour

	// displayed is how much of a token the list shows to tell them apart.
	displayed = len(PREFIX) + 4
)

var (
	ErrForbidden     = infrastructure.ErrForbidden
	ErrInvalidName   = errors.New("invalid_name")
	ErrInvalidExpiry = errors.New("invalid_expiry")
	ErrTooManyTokens = errors.New("too_many_tokens")
)

// allowedDays are the lifetimes offered; a token is never renewed, the
// developer creates a new one.
var allowedDays = map[int]bool{7: true, 15: true, 30: true}

type TokenService struct {
	store   infrastructure.AccessTokenStore
	limiter infrastructure.RateLimiter
	now     func() time.Time
}

func New(store infrastructure.AccessTokenStore, limiter infrastructure.RateLimiter, now func() time.Time) *TokenService {
	return &TokenService{store, limiter, now}
}

// Created is a new token together with its only plaintext copy.
type Created struct {
	models.AccessToken
	Token string `json:"token"`
}

// Create mints a token for the caller. days 0 means DEFAULT_DAYS. A token
// cannot mint another one.
func (srv *TokenService) Create(principal models.Principal, name string, days int) (*Created, error) {
	if principal.Anonymous() || principal.Token {
		return nil, ErrForbidden
	}

	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > MAX_NAME {
		return nil, ErrInvalidName
	}
	if days == 0 {
		days = DEFAULT_DAYS
	}
	if !allowedDays[days] {
		return nil, ErrInvalidExpiry
	}

	allowed, err := srv.limiter.Allow("pat-create:"+principal.ID, createLimit, createWindow)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, infrastructure.ErrRateLimited
	}

	existing, err := srv.store.ListAccessTokens(principal.ID)
	if err != nil {
		return nil, err
	}
	now := srv.now()
	active := 0
	for _, token := range existing {
		if token.ExpiresAt.After(now) {
			active++
		}
	}
	if active >= MAX_TOKENS {
		return nil, ErrTooManyTokens
	}

	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	raw := PREFIX + base64.RawURLEncoding.EncodeToString(secret)

	token := models.AccessToken{
		Id:        uuid.New(),
		User:      principal.ID,
		Name:      name,
		Prefix:    raw[:displayed],
		Hash:      hash(raw),
		CreatedAt: now,
		ExpiresAt: now.Add(time.Duration(days) * 24 * time.Hour),
	}
	if err := srv.store.CreateAccessToken(&token); err != nil {
		return nil, err
	}

	return &Created{AccessToken: token, Token: raw}, nil
}

func (srv *TokenService) List(principal models.Principal) ([]models.AccessToken, error) {
	if principal.Anonymous() {
		return nil, ErrForbidden
	}
	return srv.store.ListAccessTokens(principal.ID)
}

// Revoke deletes one of the caller's tokens; anyone else's is not found.
func (srv *TokenService) Revoke(principal models.Principal, id uuid.UUID) error {
	if principal.Anonymous() {
		return ErrForbidden
	}
	return srv.store.DeleteAccessToken(principal.ID, id)
}

// Resolve turns a presented token into its owner's claims. Every failure is
// ErrInvalidToken, so a caller learns nothing about why.
func (srv *TokenService) Resolve(raw string) (infrastructure.Claims, error) {
	token, err := srv.store.FindAccessTokenByHash(hash(raw))
	if err != nil {
		return infrastructure.Claims{}, infrastructure.ErrInvalidToken
	}

	now := srv.now()
	if !token.ExpiresAt.After(now) {
		return infrastructure.Claims{}, infrastructure.ErrInvalidToken
	}

	// A failed touch only loses the "last used" date; the request goes on.
	_ = srv.store.TouchAccessToken(token.Id, now)

	return infrastructure.Claims{
		Subject:     token.User,
		IssuedAt:    token.CreatedAt,
		ExpiresAt:   token.ExpiresAt,
		ID:          token.Id.String(),
		AccessToken: true,
	}, nil
}

// Verifier accepts personal access tokens and hands every other bearer to
// next, so middleware.RequireAuth takes both without knowing either.
func (srv *TokenService) Verifier(next infrastructure.TokenVerifier) infrastructure.TokenVerifier {
	return verifier{srv, next}
}

type verifier struct {
	tokens *TokenService
	next   infrastructure.TokenVerifier
}

func (v verifier) Verify(token string) (infrastructure.Claims, error) {
	if strings.HasPrefix(token, PREFIX) {
		return v.tokens.Resolve(token)
	}
	return v.next.Verify(token)
}

// The token carries 256 random bits, so a plain SHA-256 is enough: there is
// nothing to brute force, unlike a password.
func hash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
