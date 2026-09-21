package jwt

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	gojwt "github.com/golang-jwt/jwt/v5"
)

const (
	ENV_SIGNING_KEYS = "AUTH_SIGNING_KEYS"
	ISSUER           = "tiu"
)

type signingKey struct {
	id      string
	private ed25519.PrivateKey
}

// Keyring signs with its first key and verifies with any of them, so a key
// can be introduced before the previous one is retired.
type Keyring struct {
	keys []signingKey
}

var (
	_ infrastructure.TokenSigner   = (*Keyring)(nil)
	_ infrastructure.TokenVerifier = (*Keyring)(nil)
)

// New reads AUTH_SIGNING_KEYS: "<kid>:<base64 32-byte seed>[,<kid>:<seed>...]".
func New() (*Keyring, error) {
	return NewFromKeys(os.Getenv(ENV_SIGNING_KEYS))
}

func NewFromKeys(spec string) (*Keyring, error) {
	if strings.TrimSpace(spec) == "" {
		return nil, fmt.Errorf("%s is not set", ENV_SIGNING_KEYS)
	}

	ring := &Keyring{}
	for _, entry := range strings.Split(spec, ",") {
		id, encoded, ok := strings.Cut(strings.TrimSpace(entry), ":")
		if !ok || id == "" {
			return nil, fmt.Errorf("%s: expected <kid>:<base64 seed>, got %q", ENV_SIGNING_KEYS, entry)
		}

		seed, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("%s: key %s: %w", ENV_SIGNING_KEYS, id, err)
		}
		if len(seed) != ed25519.SeedSize {
			return nil, fmt.Errorf("%s: key %s: seed must be %d bytes", ENV_SIGNING_KEYS, id, ed25519.SeedSize)
		}

		ring.keys = append(ring.keys, signingKey{id: id, private: ed25519.NewKeyFromSeed(seed)})
	}

	return ring, nil
}

type tokenClaims struct {
	gojwt.RegisteredClaims
	Admin bool `json:"adm,omitempty"`
}

func (r *Keyring) Sign(c infrastructure.Claims) (string, error) {
	signer := r.keys[0]

	token := gojwt.NewWithClaims(gojwt.SigningMethodEdDSA, tokenClaims{
		RegisteredClaims: gojwt.RegisteredClaims{
			Issuer:    ISSUER,
			Subject:   c.Subject,
			ID:        c.ID,
			IssuedAt:  gojwt.NewNumericDate(c.IssuedAt),
			ExpiresAt: gojwt.NewNumericDate(c.ExpiresAt),
		},
		Admin: c.Admin,
	})
	token.Header["kid"] = signer.id

	return token.SignedString(signer.private)
}

func (r *Keyring) Verify(raw string) (infrastructure.Claims, error) {
	parsed := &tokenClaims{}

	_, err := gojwt.ParseWithClaims(raw, parsed, r.publicKey,
		gojwt.WithValidMethods([]string{gojwt.SigningMethodEdDSA.Alg()}),
		gojwt.WithIssuer(ISSUER),
		gojwt.WithExpirationRequired(),
	)
	if err != nil {
		return infrastructure.Claims{}, infrastructure.ErrInvalidToken
	}

	return infrastructure.Claims{
		Subject:   parsed.Subject,
		Admin:     parsed.Admin,
		IssuedAt:  timeOf(parsed.IssuedAt),
		ExpiresAt: timeOf(parsed.ExpiresAt),
		ID:        parsed.ID,
	}, nil
}

func (r *Keyring) publicKey(token *gojwt.Token) (any, error) {
	id, _ := token.Header["kid"].(string)
	for _, key := range r.keys {
		if key.id == id {
			return key.private.Public(), nil
		}
	}
	return nil, fmt.Errorf("unknown key %q", id)
}

func timeOf(date *gojwt.NumericDate) time.Time {
	if date == nil {
		return time.Time{}
	}
	return date.Time
}
