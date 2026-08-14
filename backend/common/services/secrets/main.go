package secrets

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/ports/crypto"
	"github.com/google/uuid"
)

const MAX_SECRET_SIZE = 8 << 10 // 8kb, generous for a token, far below a payload

var ErrForbidden = errors.New("Not allowed to manage this board's secrets")

// REFRESH_LOCK_TTL bounds how long a crashed refresh can block the others.
const REFRESH_LOCK_TTL = 30 * time.Second

type SecretsService struct {
	store      infrastructure.SecretStore
	boards     infrastructure.BoardReader
	sealer     *crypto.Sealer
	tokens     infrastructure.TokenClient
	locks      infrastructure.Locker
	handshakes infrastructure.HandshakeStore
}

func New(
	store infrastructure.SecretStore,
	boards infrastructure.BoardReader,
	sealer *crypto.Sealer,
	tokens infrastructure.TokenClient,
	locks infrastructure.Locker,
	handshakes infrastructure.HandshakeStore,
) *SecretsService {
	return &SecretsService{store, boards, sealer, tokens, locks, handshakes}
}

func (srv *SecretsService) authorize(board uuid.UUID, cognitoID string, ownerOnly bool) error {
	found, err := srv.boards.FindBoard(board)
	if err != nil {
		return err
	}

	if found.Owner == cognitoID {
		return nil
	}

	if !ownerOnly && slices.Contains(found.Collaborators, cognitoID) {
		return nil
	}

	return ErrForbidden
}

// aad binds a ciphertext to the board and name it was created under.
func aad(board uuid.UUID, name string) string {
	return board.String() + "|" + name
}

func (srv *SecretsService) Put(board uuid.UUID, cognitoID, name string, kind models.SecretKind, value string) error {
	if err := srv.authorize(board, cognitoID, true); err != nil {
		return err
	}

	if !models.ValidSecretName(name) {
		return fmt.Errorf("Secret name must match [A-Z][A-Z0-9_]*")
	}

	if value == "" {
		return fmt.Errorf("Secret value cannot be empty")
	}

	if len(value) > MAX_SECRET_SIZE {
		return fmt.Errorf("Secret value is too large")
	}

	switch kind {
	case models.SecretApiKey, models.SecretBearer, models.SecretBasic:
	default:
		return fmt.Errorf("Unsupported secret kind")
	}

	return srv.seal(board, name, kind, []byte(value))
}

func (srv *SecretsService) PutOAuth2(board uuid.UUID, cognitoID, name string, material *models.OAuth2Material) error {
	if err := srv.authorize(board, cognitoID, true); err != nil {
		return err
	}

	if !models.ValidSecretName(name) {
		return fmt.Errorf("Secret name must match [A-Z][A-Z0-9_]*")
	}

	switch material.Flow {
	case models.OAuth2ClientCredentials, models.OAuth2AuthorizationCode:
	default:
		return fmt.Errorf("Unsupported OAuth2 flow")
	}

	if material.ClientID == "" || material.ClientSecret == "" {
		return fmt.Errorf("Client id and secret are required")
	}

	if err := validEndpoint(material.TokenURL); err != nil {
		return fmt.Errorf("Token URL: %w", err)
	}

	if material.Flow == models.OAuth2AuthorizationCode && material.AuthURL != "" {
		if err := validEndpoint(material.AuthURL); err != nil {
			return fmt.Errorf("Authorization URL: %w", err)
		}
	}

	material.AccessToken = ""
	material.RefreshToken = ""
	material.TokenType = ""
	material.ExpiresAt = time.Time{}

	plaintext, err := json.Marshal(material)
	if err != nil {
		return err
	}

	return srv.seal(board, name, models.SecretOAuth2, plaintext)
}

func validEndpoint(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("is not a valid URL")
	}

	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return fmt.Errorf("must be http or https")
	}

	if parsed.Host == "" {
		return fmt.Errorf("must be absolute")
	}

	if parsed.User != nil {
		return fmt.Errorf("must not embed credentials")
	}

	return nil
}

func (srv *SecretsService) List(board uuid.UUID, cognitoID string) ([]models.SecretMeta, error) {
	if err := srv.authorize(board, cognitoID, false); err != nil {
		return nil, err
	}

	stored, err := srv.store.ListSecrets(board)
	if err != nil {
		return nil, err
	}

	metas := make([]models.SecretMeta, 0, len(stored))
	for _, s := range stored {
		metas = append(metas, s.Meta())
	}

	return metas, nil
}

func (srv *SecretsService) Delete(board uuid.UUID, cognitoID, name string) error {
	if err := srv.authorize(board, cognitoID, true); err != nil {
		return err
	}

	return srv.store.DeleteSecret(board, name)
}

// store seals a value and writes it, preserving the board + name binding.
func (srv *SecretsService) seal(board uuid.UUID, name string, kind models.SecretKind, plaintext []byte) error {
	sealed, err := srv.sealer.Seal(aad(board, name), plaintext)
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	return srv.store.UpsertSecret(&models.Secret{
		Id:         uuid.New(),
		Board:      board,
		Name:       name,
		Kind:       kind,
		Ciphertext: sealed.Ciphertext,
		Nonce:      sealed.Nonce,
		KeyVersion: sealed.KeyVersion,
		CreatedAt:  now,
		UpdatedAt:  now,
	})
}

func (srv *SecretsService) unseal(s *models.Secret) ([]byte, error) {
	return srv.sealer.Open(aad(s.Board, s.Name), &crypto.Sealed{
		Ciphertext: s.Ciphertext,
		Nonce:      s.Nonce,
		KeyVersion: s.KeyVersion,
	})
}

func (srv *SecretsService) refresh(s *models.Secret, material *models.OAuth2Material) error {
	key := "oauth-refresh:" + s.Board.String() + ":" + s.Name

	held, err := srv.locks.Acquire(key, REFRESH_LOCK_TTL)
	if err != nil {
		return err
	}

	if !held {
		fresh, err := srv.store.FindSecrets(s.Board, []string{s.Name})
		if err != nil {
			return err
		}
		if len(fresh) == 0 {
			return fmt.Errorf("Credential disappeared during refresh")
		}

		plaintext, err := srv.unseal(&fresh[0])
		if err != nil {
			return err
		}
		if err := json.Unmarshal(plaintext, material); err != nil {
			return err
		}
		if material.NeedsRefresh() {
			return fmt.Errorf("Another refresh is in progress")
		}

		return nil
	}

	defer srv.locks.Release(key)

	if err := srv.tokens.Fetch(material); err != nil {
		return err
	}

	plaintext, err := json.Marshal(material)
	if err != nil {
		return err
	}

	return srv.seal(s.Board, s.Name, models.SecretOAuth2, plaintext)
}

func (srv *SecretsService) Resolve(board uuid.UUID, names []string) (map[string]string, error) {
	if len(names) == 0 {
		return nil, nil
	}

	stored, err := srv.store.FindSecrets(board, names)
	if err != nil {
		return nil, err
	}

	resolved := make(map[string]string, len(stored))
	for _, s := range stored {
		value, err := srv.unseal(&s)
		if err != nil {
			return nil, err
		}

		if s.Kind != models.SecretOAuth2 {
			resolved["$"+s.Name] = string(value)
			continue
		}

		var material models.OAuth2Material
		if err := json.Unmarshal(value, &material); err != nil {
			return nil, err
		}

		if material.NeedsRefresh() {
			if err := srv.refresh(&s, &material); err != nil {
				return nil, err
			}
		}

		resolved["$"+s.Name] = material.Header()
	}

	return resolved, nil
}
