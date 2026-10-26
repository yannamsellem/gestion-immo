package memory

import (
	"context"
	"sync"

	"github.com/yannamsellem/gestion-immo/internal/modules/document/domain"
)

// DocumentTemplateRepo is a thread-safe in-memory implementation of domain.DocumentTemplateRepository for tests.
type DocumentTemplateRepo struct {
	mu        sync.RWMutex
	templates map[string]domain.DocumentTemplate
	tplOrder  []string
}

// NewDocumentTemplateRepo constructs an empty thread-safe in-memory DocumentTemplateRepo.
func NewDocumentTemplateRepo() *DocumentTemplateRepo {
	return &DocumentTemplateRepo{
		templates: make(map[string]domain.DocumentTemplate),
	}
}

// SaveTemplate stores a DocumentTemplate in memory.
func (r *DocumentTemplateRepo) SaveTemplate(_ context.Context, tpl domain.DocumentTemplate) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.templates[tpl.ID]; !exists {
		r.tplOrder = append(r.tplOrder, tpl.ID)
	}
	r.templates[tpl.ID] = tpl
	return nil
}

// FindTemplateByID retrieves a DocumentTemplate by ID from memory.
func (r *DocumentTemplateRepo) FindTemplateByID(_ context.Context, id string) (*domain.DocumentTemplate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tpl, ok := r.templates[id]
	if !ok {
		return nil, domain.ErrTemplateNotFound
	}
	copyTpl := tpl
	return &copyTpl, nil
}

// ListTemplates returns all DocumentTemplates in insertion order.
func (r *DocumentTemplateRepo) ListTemplates(_ context.Context) ([]domain.DocumentTemplate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]domain.DocumentTemplate, 0, len(r.tplOrder))
	for _, id := range r.tplOrder {
		out = append(out, r.templates[id])
	}
	return out, nil
}
