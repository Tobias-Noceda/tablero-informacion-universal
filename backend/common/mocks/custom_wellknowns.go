package mocks

import (
	"slices"
	"strings"
	"sync"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
)

// MemoryCustomWellKnownStore is an in-memory infrastructure.CustomWellKnownStore.
type MemoryCustomWellKnownStore struct {
	mu   sync.Mutex
	defs map[string]models.CustomWellKnown
}

var _ infrastructure.CustomWellKnownStore = (*MemoryCustomWellKnownStore)(nil)

func NewMemoryCustomWellKnownStore(defs ...models.CustomWellKnown) *MemoryCustomWellKnownStore {
	m := &MemoryCustomWellKnownStore{defs: map[string]models.CustomWellKnown{}}
	for _, c := range defs {
		m.defs[c.Key] = c
	}
	return m
}

func (m *MemoryCustomWellKnownStore) FindCustomWellKnown(key string) (*models.CustomWellKnown, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	c, ok := m.defs[key]
	if !ok {
		return nil, infrastructure.ErrCustomWellKnownNotFound
	}
	return &c, nil
}

func (m *MemoryCustomWellKnownStore) FindCustomWellKnowns(keys []string) ([]models.CustomWellKnown, error) {
	return m.filter(func(c models.CustomWellKnown) bool { return slices.Contains(keys, c.Key) }), nil
}

func (m *MemoryCustomWellKnownStore) ListCustomWellKnowns(scopes ...models.CustomWellKnownScope) ([]models.CustomWellKnown, error) {
	return m.filter(func(c models.CustomWellKnown) bool { return slices.Contains(scopes, c.Scope) }), nil
}

func (m *MemoryCustomWellKnownStore) CountCustomWellKnowns(scope models.CustomWellKnownScope) (int64, error) {
	return int64(len(m.filter(func(c models.CustomWellKnown) bool { return c.Scope == scope }))), nil
}

func (m *MemoryCustomWellKnownStore) PutCustomWellKnown(c *models.CustomWellKnown) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.defs[c.Key] = *c
	return nil
}

func (m *MemoryCustomWellKnownStore) DeleteCustomWellKnown(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.defs[key]; !ok {
		return infrastructure.ErrCustomWellKnownNotFound
	}
	delete(m.defs, key)
	return nil
}

func (m *MemoryCustomWellKnownStore) filter(keep func(models.CustomWellKnown) bool) []models.CustomWellKnown {
	m.mu.Lock()
	defer m.mu.Unlock()

	list := []models.CustomWellKnown{}
	for _, c := range m.defs {
		if keep(c) {
			list = append(list, c)
		}
	}
	slices.SortFunc(list, func(a, b models.CustomWellKnown) int {
		if n := strings.Compare(a.Name, b.Name); n != 0 {
			return n
		}
		return strings.Compare(a.Key, b.Key)
	})
	return list
}
