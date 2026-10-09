package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

var (
	ErrGoogleDisabled   = errors.New("google sign-in is not configured")
	ErrInvalidNext      = errors.New("next must be a path on this site")
	ErrInvalidState     = errors.New("invalid or expired sign-in state")
	ErrIdentityProvider = infrastructure.ErrIdentityProvider
)

const (
	GOOGLE_CALLBACK_PATH = "/auth/google/callback"
	SIGNIN_TTL           = 10 * time.Minute
)

// GoogleSignIn is where to send the browser and the value that ties the
// callback to that same browser (the controller keeps it in a cookie).
type GoogleSignIn struct {
	URL     string
	Binding string
}

type pendingSignIn struct {
	Verifier    string `json:"verifier"`
	Nonce       string `json:"nonce"`
	RedirectURI string `json:"redirect_uri"`
	Next        string `json:"next"`
}

func (srv *AuthService) GoogleStart(origin, next string) (*GoogleSignIn, error) {
	if srv.google == nil {
		return nil, ErrGoogleDisabled
	}
	next, ok := relativePath(next)
	if !ok {
		return nil, ErrInvalidNext
	}

	state, err := randomToken()
	if err != nil {
		return nil, err
	}
	nonce, err := randomToken()
	if err != nil {
		return nil, err
	}
	verifier, err := randomToken()
	if err != nil {
		return nil, err
	}

	pending := pendingSignIn{Verifier: verifier, Nonce: nonce, RedirectURI: origin + GOOGLE_CALLBACK_PATH, Next: next}
	encoded, err := json.Marshal(pending)
	if err != nil {
		return nil, err
	}
	binding := hashToken(state)
	if err := srv.tokens.Put(tokenKey("signin", binding), encoded, SIGNIN_TTL); err != nil {
		return nil, err
	}

	challenge := sha256.Sum256([]byte(verifier))
	target, err := srv.google.AuthURL(state, nonce, base64.RawURLEncoding.EncodeToString(challenge[:]), pending.RedirectURI)
	if err != nil {
		return nil, err
	}

	return &GoogleSignIn{URL: target, Binding: binding}, nil
}

// GoogleCallback finishes a sign-in started by GoogleStart in the same
// browser and returns the session plus where the user wanted to go.
func (srv *AuthService) GoogleCallback(state, code, binding string) (*Tokens, string, error) {
	if srv.google == nil {
		return nil, "", ErrGoogleDisabled
	}
	if state == "" || subtle.ConstantTimeCompare([]byte(hashToken(state)), []byte(binding)) != 1 {
		return nil, "", ErrInvalidState
	}

	raw, err := srv.tokens.Take(tokenKey("signin", binding))
	if err != nil {
		return nil, "", ErrInvalidState
	}
	var pending pendingSignIn
	if err := json.Unmarshal(raw, &pending); err != nil {
		return nil, "", ErrInvalidState
	}

	identity, err := srv.google.Complete(code, pending.RedirectURI, pending.Verifier, pending.Nonce)
	if err != nil {
		return nil, "", err
	}
	if !identity.EmailVerified {
		return nil, "", ErrEmailNotVerified
	}

	user, err := srv.linkGoogle(identity)
	if err != nil {
		return nil, "", err
	}

	tokens, err := srv.issue(user, uuid.NewString())
	if err != nil {
		return nil, "", err
	}
	return tokens, pending.Next, nil
}

// linkGoogle finds the account a Google identity belongs to, by subject
// first and by verified email second, creating it when there is none.
func (srv *AuthService) linkGoogle(external *infrastructure.ExternalIdentity) (*models.User, error) {
	google := models.Identity{Provider: models.IdentityGoogle, Subject: external.Subject}

	user, err := srv.users.FindUserByIdentity(models.IdentityGoogle, external.Subject)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, infrastructure.ErrUserNotFound) {
		return nil, err
	}

	email := models.NormalizeEmail(external.Email)
	if !models.ValidEmail(email) {
		return nil, fmt.Errorf("%w: the id_token carries no usable email", ErrIdentityProvider)
	}

	user, err = srv.users.FindUserByEmail(email)
	switch {
	case errors.Is(err, infrastructure.ErrUserNotFound):
		return srv.createGoogleUser(email, external, google)
	case err != nil:
		return nil, err
	case user.EmailVerified:
		return user, srv.addGoogle(user, external, google)
	default:
		return user, srv.takeOverUnverified(user, external, google)
	}
}

func (srv *AuthService) createGoogleUser(email string, external *infrastructure.ExternalIdentity, google models.Identity) (*models.User, error) {
	name := strings.TrimSpace(external.Name)
	if name == "" {
		name, _, _ = strings.Cut(email, "@")
	}

	now := srv.now()
	user := &models.User{
		Id:            uuid.New(),
		Email:         email,
		EmailVerified: true,
		Name:          name,
		Picture:       external.Picture,
		Admin:         srv.admins[email],
		Identities:    []models.Identity{google},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := srv.users.CreateUser(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (srv *AuthService) addGoogle(user *models.User, external *infrastructure.ExternalIdentity, google models.Identity) error {
	user.Identities = withIdentity(user.Identities, google)
	if err := srv.users.ReplaceIdentities(user.Id, user.Identities); err != nil {
		return err
	}
	if user.Picture != "" || external.Picture == "" {
		return nil
	}
	user.Picture = external.Picture
	return srv.users.UpdateUser(user.Id, map[string]any{"picture": user.Picture})
}

// takeOverUnverified hands an address nobody proved to the Google account
// that does own it: whatever the registration set up is dropped.
func (srv *AuthService) takeOverUnverified(user *models.User, external *infrastructure.ExternalIdentity, google models.Identity) error {
	user.Identities = []models.Identity{google}
	if err := srv.users.ReplaceIdentities(user.Id, user.Identities); err != nil {
		return err
	}

	user.EmailVerified = true
	set := map[string]any{"emailverified": true, "picture": external.Picture}
	user.Picture = external.Picture
	if name := strings.TrimSpace(external.Name); name != "" {
		user.Name = name
		set["name"] = name
	}
	if err := srv.users.UpdateUser(user.Id, set); err != nil {
		return err
	}

	return srv.sessions.RevokeUser(user.Id.String())
}

// relativePath accepts only a path on this origin, so the post sign-in
// redirect can never leave the site. Browsers read "\" as "/" and drop tabs
// and newlines, which would turn "/\evil" or "/\t/evil" into "//evil".
func relativePath(next string) (string, bool) {
	if next == "" {
		return "/", true
	}
	if !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.ContainsRune(next, '\\') {
		return "", false
	}
	for _, r := range next {
		if r < 0x20 || r == 0x7f {
			return "", false
		}
	}
	parsed, err := url.Parse(next)
	if err != nil || parsed.Scheme != "" || parsed.Host != "" {
		return "", false
	}
	return next, true
}
