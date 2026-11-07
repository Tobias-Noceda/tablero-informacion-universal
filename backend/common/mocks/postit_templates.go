package mocks

import (
	"slices"
	"strings"
	"sync"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
)

// MemoryPostitTemplateStore is an in-memory infrastructure.PostitTemplateStore.
type MemoryPostitTemplateStore struct {
	mu   sync.Mutex
	defs map[string]models.PostitTemplate
}

var _ infrastructure.PostitTemplateStore = (*MemoryPostitTemplateStore)(nil)

func NewMemoryPostitTemplateStore(defs ...models.PostitTemplate) *MemoryPostitTemplateStore {
	m := &MemoryPostitTemplateStore{defs: map[string]models.PostitTemplate{}}
	for _, c := range defs {
		m.defs[c.Key] = c
	}
	return m
}

func (m *MemoryPostitTemplateStore) FindPostitTemplate(key string) (*models.PostitTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	c, ok := m.defs[key]
	if !ok {
		return nil, infrastructure.ErrPostitTemplateNotFound
	}
	return &c, nil
}

func (m *MemoryPostitTemplateStore) FindPostitTemplates(keys []string) ([]models.PostitTemplate, error) {
	return m.filter(func(c models.PostitTemplate) bool { return slices.Contains(keys, c.Key) }), nil
}

func (m *MemoryPostitTemplateStore) ListPostitTemplates(scopes ...models.PostitTemplateScope) ([]models.PostitTemplate, error) {
	return m.filter(func(c models.PostitTemplate) bool { return slices.Contains(scopes, c.Scope) }), nil
}

func (m *MemoryPostitTemplateStore) CountPostitTemplates(scope models.PostitTemplateScope) (int64, error) {
	return int64(len(m.filter(func(c models.PostitTemplate) bool { return c.Scope == scope }))), nil
}

func (m *MemoryPostitTemplateStore) PutPostitTemplate(c *models.PostitTemplate) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.defs[c.Key] = *c
	return nil
}

func (m *MemoryPostitTemplateStore) DeletePostitTemplate(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.defs[key]; !ok {
		return infrastructure.ErrPostitTemplateNotFound
	}
	delete(m.defs, key)
	return nil
}

func (m *MemoryPostitTemplateStore) filter(keep func(models.PostitTemplate) bool) []models.PostitTemplate {
	m.mu.Lock()
	defer m.mu.Unlock()

	list := []models.PostitTemplate{}
	for _, c := range m.defs {
		if keep(c) {
			list = append(list, c)
		}
	}
	slices.SortFunc(list, func(a, b models.PostitTemplate) int {
		if n := strings.Compare(a.Name, b.Name); n != 0 {
			return n
		}
		return strings.Compare(a.Key, b.Key)
	})
	return list
}
