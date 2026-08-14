package secrets

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

// HANDSHAKE_TTL is how long a user has to finish consenting before the
// pending authorization is discarded.
const HANDSHAKE_TTL = 10 * time.Minute

type pending struct {
	Board    uuid.UUID `json:"board"`
	Name     string    `json:"name"`
	Verifier string    `json:"verifier"`
	Redirect string    `json:"redirect"`
}

func randomURLSafe(bytes int) (string, error) {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (srv *SecretsService) Authorize(board uuid.UUID, cognitoID, name, redirectURI string) (string, error) {
	if err := srv.authorize(board, cognitoID, true); err != nil {
		return "", err
	}

	material, _, err := srv.material(board, name)
	if err != nil {
		return "", err
	}

	if material.Flow != models.OAuth2AuthorizationCode {
		return "", fmt.Errorf("Credential does not use the authorization code flow")
	}

	if material.AuthURL == "" {
		return "", fmt.Errorf("Credential has no authorization URL")
	}

	if err := validEndpoint(redirectURI); err != nil {
		return "", fmt.Errorf("Redirect URI: %w", err)
	}

	state, err := randomURLSafe(32)
	if err != nil {
		return "", err
	}

	verifier, err := randomURLSafe(32)
	if err != nil {
		return "", err
	}

	handshake, err := json.Marshal(&pending{
		Board:    board,
		Name:     name,
		Verifier: verifier,
		Redirect: redirectURI,
	})
	if err != nil {
		return "", err
	}

	if err := srv.handshakes.Put(handshakeKey(state), handshake, HANDSHAKE_TTL); err != nil {
		return "", err
	}

	sum := sha256.Sum256([]byte(verifier))

	target, err := url.Parse(material.AuthURL)
	if err != nil {
		return "", err
	}

	query := target.Query()
	query.Set("response_type", "code")
	query.Set("client_id", material.ClientID)
	query.Set("redirect_uri", redirectURI)
	query.Set("state", state)
	query.Set("code_challenge", base64.RawURLEncoding.EncodeToString(sum[:]))
	query.Set("code_challenge_method", "S256")
	if material.Scopes != "" {
		query.Set("scope", material.Scopes)
	}
	target.RawQuery = query.Encode()

	return target.String(), nil
}

func (srv *SecretsService) Callback(state, code string) error {
	if state == "" || code == "" {
		return fmt.Errorf("Missing state or code")
	}

	raw, err := srv.handshakes.Take(handshakeKey(state))
	if err != nil {
		return fmt.Errorf("Unknown or expired authorization")
	}

	var handshake pending
	if err := json.Unmarshal(raw, &handshake); err != nil {
		return err
	}

	material, secret, err := srv.material(handshake.Board, handshake.Name)
	if err != nil {
		return err
	}

	if err := srv.tokens.Exchange(material, code, handshake.Redirect, handshake.Verifier); err != nil {
		return err
	}

	plaintext, err := json.Marshal(material)
	if err != nil {
		return err
	}

	return srv.seal(secret.Board, secret.Name, models.SecretOAuth2, plaintext, secret.Flow, true)
}

func handshakeKey(state string) string {
	return "oauth-handshake:" + state
}

// material loads and decrypts an OAuth2 credential by name.
func (srv *SecretsService) material(board uuid.UUID, name string) (*models.OAuth2Material, *models.Secret, error) {
	stored, err := srv.store.FindSecrets(board, []string{name})
	if err != nil {
		return nil, nil, err
	}

	if len(stored) == 0 {
		return nil, nil, fmt.Errorf("Credential not found")
	}

	secret := stored[0]
	if secret.Kind != models.SecretOAuth2 {
		return nil, nil, fmt.Errorf("Credential is not an OAuth2 credential")
	}

	plaintext, err := srv.unseal(&secret)
	if err != nil {
		return nil, nil, err
	}

	var material models.OAuth2Material
	if err := json.Unmarshal(plaintext, &material); err != nil {
		return nil, nil, err
	}

	return &material, &secret, nil
}
