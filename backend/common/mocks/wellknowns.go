package mocks

import (
	"slices"
	"strings"
	"sync"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
)

// MemoryWellKnownStore is an in-memory infrastructure.WellKnownStore with the
// same tombstone and seed semantics as the Mongo one.
type MemoryWellKnownStore struct {
	mu   sync.Mutex
	defs map[string]models.WellKnown
}

var _ infrastructure.WellKnownStore = (*MemoryWellKnownStore)(nil)

func NewMemoryWellKnownStore(defs ...models.WellKnown) *MemoryWellKnownStore {
	m := &MemoryWellKnownStore{defs: map[string]models.WellKnown{}}
	for _, wk := range defs {
		m.defs[wk.Key] = wk
	}
	return m
}

func (m *MemoryWellKnownStore) FindWellKnown(key string) (*models.WellKnown, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	wk, ok := m.defs[key]
	if !ok {
		return nil, infrastructure.ErrWellKnownNotFound
	}
	return &wk, nil
}

func (m *MemoryWellKnownStore) ListWellKnowns() ([]models.WellKnown, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	wks := []models.WellKnown{}
	for _, wk := range m.defs {
		if !wk.Deleted {
			wks = append(wks, wk)
		}
	}
	slices.SortFunc(wks, func(a, b models.WellKnown) int {
		if a.Order != b.Order {
			return a.Order - b.Order
		}
		return strings.Compare(a.Key, b.Key)
	})
	return wks, nil
}

func (m *MemoryWellKnownStore) PutWellKnown(wk *models.WellKnown) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	stored := *wk
	stored.Deleted = false
	m.defs[wk.Key] = stored
	return nil
}

func (m *MemoryWellKnownStore) DeleteWellKnown(key string) error {
	return m.setDeleted(key, true)
}

func (m *MemoryWellKnownStore) RestoreWellKnown(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	wk, ok := m.defs[key]
	if !ok {
		return infrastructure.ErrWellKnownNotFound
	}
	if !wk.Deleted {
		return infrastructure.ErrWellKnownNotDeleted
	}
	wk.Deleted = false
	m.defs[key] = wk
	return nil
}

func (m *MemoryWellKnownStore) setDeleted(key string, deleted bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	wk, ok := m.defs[key]
	if !ok || wk.Deleted == deleted {
		return infrastructure.ErrWellKnownNotFound
	}
	wk.Deleted = deleted
	m.defs[key] = wk
	return nil
}

func (m *MemoryWellKnownStore) SeedWellKnowns(defs []models.WellKnown) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, wk := range defs {
		if _, ok := m.defs[wk.Key]; !ok {
			m.defs[wk.Key] = wk
		}
	}
	return nil
}
