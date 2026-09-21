package mocks

import (
	"sync"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

// MemoryKeyStore is an in-memory infrastructure.KeyStore with the same
// uniqueness rule as the Mongo partial index: one active key per scope.
type MemoryKeyStore struct {
	mu   sync.Mutex
	keys []models.DataKey
}

var _ infrastructure.KeyStore = (*MemoryKeyStore)(nil)

func (m *MemoryKeyStore) FindActiveKey(scope models.SecretScope) (*models.DataKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, key := range m.keys {
		if key.Scope == scope && key.Active {
			found := key
			return &found, nil
		}
	}
	return nil, infrastructure.ErrKeyNotFound
}

func (m *MemoryKeyStore) FindKey(id uuid.UUID) (*models.DataKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, key := range m.keys {
		if key.Id == id {
			found := key
			return &found, nil
		}
	}
	return nil, infrastructure.ErrKeyNotFound
}

func (m *MemoryKeyStore) InsertKey(key *models.DataKey) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if key.Active {
		for _, existing := range m.keys {
			if existing.Scope == key.Scope && existing.Active {
				return infrastructure.ErrActiveKeyExists
			}
		}
	}
	m.keys = append(m.keys, *key)
	return nil
}

func (m *MemoryKeyStore) RetireKey(id uuid.UUID, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.keys {
		if m.keys[i].Id == id {
			m.keys[i].Active = false
			m.keys[i].RetiredAt = &at
			return nil
		}
	}
	return infrastructure.ErrKeyNotFound
}

func (m *MemoryKeyStore) Rewrap(id uuid.UUID, wrapped, nonce []byte, kekVersion int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.keys {
		if m.keys[i].Id == id {
			m.keys[i].WrappedKey = wrapped
			m.keys[i].Nonce = nonce
			m.keys[i].KEKVersion = kekVersion
			return nil
		}
	}
	return infrastructure.ErrKeyNotFound
}

func (m *MemoryKeyStore) DeleteRetiredKeys(scope models.SecretScope) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.keys = m.filter(func(key models.DataKey) bool {
		return key.Scope != scope || key.Active
	})
	return nil
}

func (m *MemoryKeyStore) DeleteKeys(scope models.SecretScope) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.keys = m.filter(func(key models.DataKey) bool {
		return key.Scope != scope
	})
	return nil
}

func (m *MemoryKeyStore) ListKeys() ([]models.DataKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]models.DataKey, len(m.keys))
	copy(out, m.keys)
	return out, nil
}

func (m *MemoryKeyStore) filter(keep func(models.DataKey) bool) []models.DataKey {
	kept := m.keys[:0]
	for _, key := range m.keys {
		if keep(key) {
			kept = append(kept, key)
		}
	}
	return kept
}
