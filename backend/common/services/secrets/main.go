package secrets

import (
	"fmt"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/ports/crypto"
	"github.com/google/uuid"
)

const MAX_SECRET_SIZE = 8 << 10 // 8kb, generous for a token, far below a payload

type SecretsService struct {
	store  infrastructure.SecretStore
	sealer *crypto.Sealer
}

func New(store infrastructure.SecretStore, sealer *crypto.Sealer) *SecretsService {
	return &SecretsService{store, sealer}
}

// aad binds a ciphertext to the board and name it was created under.
func aad(board uuid.UUID, name string) string {
	return board.String() + "|" + name
}

func (srv *SecretsService) Put(board uuid.UUID, name string, kind models.SecretKind, value string) error {
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

	sealed, err := srv.sealer.Seal(aad(board, name), []byte(value))
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

func (srv *SecretsService) List(board uuid.UUID) ([]models.SecretMeta, error) {
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

func (srv *SecretsService) Delete(board uuid.UUID, name string) error {
	return srv.store.DeleteSecret(board, name)
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
		value, err := srv.sealer.Open(aad(s.Board, s.Name), &crypto.Sealed{
			Ciphertext: s.Ciphertext,
			Nonce:      s.Nonce,
			KeyVersion: s.KeyVersion,
		})
		if err != nil {
			return nil, err
		}

		resolved["$"+s.Name] = string(value)
	}

	return resolved, nil
}
