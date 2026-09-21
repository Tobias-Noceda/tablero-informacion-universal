package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

const DATA_KEY_SIZE = 32

type Sealed struct {
	Ciphertext []byte
	Nonce      []byte
	KeyID      uuid.UUID
}

// Keyring seals with one data key per scope and keeps those data keys
// wrapped by the master key. Rotating the master key only rewraps data keys;
// rotating a data key only touches its own scope.
type Keyring struct {
	kek   *Sealer
	store infrastructure.KeyStore
}

func NewKeyring(kek *Sealer, store infrastructure.KeyStore) *Keyring {
	return &Keyring{kek, store}
}

// wrapAAD binds a wrapped data key to its scope, so a key document moved to
// another scope in the database does not unwrap there.
func wrapAAD(scope models.SecretScope) string {
	return "dek|" + scope.Key()
}

func (k *Keyring) Seal(scope models.SecretScope, aad string, plaintext []byte) (*Sealed, error) {
	key, err := k.activeKey(scope)
	if err != nil {
		return nil, err
	}

	aead, err := k.unwrap(key)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	return &Sealed{
		Ciphertext: aead.Seal(nil, nonce, plaintext, []byte(aad)),
		Nonce:      nonce,
		KeyID:      key.Id,
	}, nil
}

// Open decrypts and reports whether the ciphertext is under a retired key,
// so the caller can reseal it under the active one.
func (k *Keyring) Open(scope models.SecretScope, aad string, sealed *Sealed) (plaintext []byte, stale bool, err error) {
	key, err := k.store.FindKey(sealed.KeyID)
	if err != nil {
		return nil, false, err
	}

	if key.Scope != scope {
		return nil, false, infrastructure.ErrKeyNotFound
	}

	aead, err := k.unwrap(key)
	if err != nil {
		return nil, false, err
	}

	plaintext, err = aead.Open(nil, sealed.Nonce, sealed.Ciphertext, []byte(aad))
	if err != nil {
		return nil, false, fmt.Errorf("could not decrypt secret")
	}

	return plaintext, !key.Active, nil
}

func (k *Keyring) Rotate(scope models.SecretScope) (*models.DataKey, error) {
	current, err := k.store.FindActiveKey(scope)
	if err != nil && !errors.Is(err, infrastructure.ErrKeyNotFound) {
		return nil, err
	}

	if current != nil {
		if err := k.store.RetireKey(current.Id, time.Now().UTC()); err != nil {
			return nil, err
		}
	}

	return k.provision(scope)
}

func (k *Keyring) Shred(scope models.SecretScope) error {
	return k.store.DeleteKeys(scope)
}

func (k *Keyring) Prune(scope models.SecretScope) error {
	return k.store.DeleteRetiredKeys(scope)
}

// Rewrap brings every data key under the current master key version and
// returns how many it touched.
func (k *Keyring) Rewrap() (int, error) {
	keys, err := k.store.ListKeys()
	if err != nil {
		return 0, err
	}

	rewrapped := 0
	for _, key := range keys {
		if !k.kek.NeedsRotation(&Wrapped{KeyVersion: key.KEKVersion}) {
			continue
		}

		raw, err := k.kek.Open(wrapAAD(key.Scope), &Wrapped{
			Ciphertext: key.WrappedKey,
			Nonce:      key.Nonce,
			KeyVersion: key.KEKVersion,
		})
		if err != nil {
			return rewrapped, fmt.Errorf("key %s: %w", key.Id, err)
		}

		wrapped, err := k.kek.Seal(wrapAAD(key.Scope), raw)
		if err != nil {
			return rewrapped, err
		}

		if err := k.store.Rewrap(key.Id, wrapped.Ciphertext, wrapped.Nonce, wrapped.KeyVersion); err != nil {
			return rewrapped, err
		}
		rewrapped++
	}

	return rewrapped, nil
}

func (k *Keyring) activeKey(scope models.SecretScope) (*models.DataKey, error) {
	key, err := k.store.FindActiveKey(scope)
	if err == nil {
		return key, nil
	}
	if !errors.Is(err, infrastructure.ErrKeyNotFound) {
		return nil, err
	}

	fresh, err := k.provision(scope)
	if errors.Is(err, infrastructure.ErrActiveKeyExists) {
		return k.store.FindActiveKey(scope)
	}

	return fresh, err
}

func (k *Keyring) provision(scope models.SecretScope) (*models.DataKey, error) {
	raw := make([]byte, DATA_KEY_SIZE)
	if _, err := rand.Read(raw); err != nil {
		return nil, err
	}

	wrapped, err := k.kek.Seal(wrapAAD(scope), raw)
	if err != nil {
		return nil, err
	}

	key := &models.DataKey{
		Id:         uuid.New(),
		Scope:      scope,
		WrappedKey: wrapped.Ciphertext,
		Nonce:      wrapped.Nonce,
		KEKVersion: wrapped.KeyVersion,
		Active:     true,
		CreatedAt:  time.Now().UTC(),
	}

	if err := k.store.InsertKey(key); err != nil {
		return nil, err
	}

	return key, nil
}

func (k *Keyring) unwrap(key *models.DataKey) (cipher.AEAD, error) {
	raw, err := k.kek.Open(wrapAAD(key.Scope), &Wrapped{
		Ciphertext: key.WrappedKey,
		Nonce:      key.Nonce,
		KeyVersion: key.KEKVersion,
	})
	if err != nil {
		return nil, fmt.Errorf("could not unwrap data key: %w", err)
	}

	block, err := aes.NewCipher(raw)
	if err != nil {
		return nil, err
	}

	return cipher.NewGCM(block)
}

func (k *Keyring) List() ([]models.DataKey, error) {
	return k.store.ListKeys()
}
