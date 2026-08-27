package device

import (
	"fmt"
	"sync"

	"github.com/dyl-03/shadow/internal/model"
)

// TemplateStore resolves device templates by name.
type TemplateStore struct {
	mu        sync.RWMutex
	templates map[string]*model.Template
}

func NewTemplateStore() *TemplateStore {
	return &TemplateStore{templates: make(map[string]*model.Template)}
}

func (s *TemplateStore) Register(t *model.Template) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.templates[t.Name]; ok {
		return fmt.Errorf("template %s already registered", t.Name)
	}
	cp := *t
	s.templates[t.Name] = &cp
	return nil
}

func (s *TemplateStore) Resolve(name string) *model.Template {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.templates[name]
	if !ok {
		return nil
	}
	cp := *t
	return &cp
}
