package secrets

import (
	"log"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
)

// RESEAL_LOCK_TTL bounds how long a crashed reseal can block the others.
const RESEAL_LOCK_TTL = 30 * time.Second

var _ infrastructure.ScopePurger = (*SecretsService)(nil)

// migrate reseals a secret found under a retired key. Best effort: the value
// was already recovered, so a failure here only delays the migration.
func (srv *SecretsService) migrate(s *models.Secret, plaintext []byte) {
	key := "reseal:" + s.Scope.Key() + ":" + s.Name

	token, held, err := srv.locks.Acquire(key, RESEAL_LOCK_TTL)
	if err != nil || !held {
		return
	}
	defer srv.locks.Release(key, token)

	if err := srv.seal(s.Scope, s.Name, s.Kind, plaintext, s.Flow, s.Authorized); err != nil {
		log.Printf("secret %s/%s: migrating to the active key failed: %v", s.Scope.Key(), s.Name, err)
	}
}

// Reseal moves every secret of a scope under its active key and drops the
// retired ones. Returns how many secrets it touched.
func (srv *SecretsService) Reseal(scope models.SecretScope) (int, error) {
	stored, err := srv.store.ListSecrets(scope)
	if err != nil {
		return 0, err
	}

	resealed := 0
	for _, s := range stored {
		plaintext, stale, err := srv.open(&s)
		if err != nil {
			return resealed, err
		}
		if !stale {
			continue
		}

		if err := srv.seal(s.Scope, s.Name, s.Kind, plaintext, s.Flow, s.Authorized); err != nil {
			return resealed, err
		}
		resealed++
	}

	return resealed, srv.keyring.Prune(scope)
}

func (srv *SecretsService) RotateScope(scope models.SecretScope) error {
	if _, err := srv.keyring.Rotate(scope); err != nil {
		return err
	}

	_, err := srv.Reseal(scope)
	return err
}

func (srv *SecretsService) RotateAll() error {
	keys, err := srv.keyring.List()
	if err != nil {
		return err
	}

	for _, key := range keys {
		if !key.Active {
			continue
		}
		if err := srv.RotateScope(key.Scope); err != nil {
			return err
		}
	}

	return nil
}

func (srv *SecretsService) Rewrap() (int, error) {
	return srv.keyring.Rewrap()
}

func (srv *SecretsService) Purge(scope models.SecretScope) error {
	if err := srv.store.DeleteSecrets(scope); err != nil {
		return err
	}

	return srv.keyring.Shred(scope)
}

func (srv *SecretsService) ListKeys(principal models.Principal) ([]models.DataKey, error) {
	if err := srv.policy.CanView(principal, models.SystemScope); err != nil {
		return nil, err
	}

	keys, err := srv.keyring.List()
	if err != nil {
		return nil, err
	}

	for i := range keys {
		keys[i].WrappedKey = nil
		keys[i].Nonce = nil
	}

	return keys, nil
}
