package memory

import (
	"context"
	"sync"

	"github.com/yannamsellem/gestion-immo/internal/modules/inspection/domain"
)

// InspectionTemplateRepo is a thread-safe in-memory implementation of domain.InspectionTemplateRepository for tests.
type InspectionTemplateRepo struct {
	mu        sync.RWMutex
	templates map[string]domain.InspectionTemplate
	tplOrder  []string
}

// NewInspectionTemplateRepo constructs an empty thread-safe in-memory InspectionTemplateRepo.
func NewInspectionTemplateRepo() *InspectionTemplateRepo {
	return &InspectionTemplateRepo{
		templates: make(map[string]domain.InspectionTemplate),
	}
}

// SaveTemplate stores an InspectionTemplate in memory.
func (r *InspectionTemplateRepo) SaveTemplate(_ context.Context, tpl domain.InspectionTemplate) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	copyTpl := cloneTemplate(tpl)
	if _, exists := r.templates[tpl.ID]; !exists {
		r.tplOrder = append(r.tplOrder, tpl.ID)
	}
	r.templates[tpl.ID] = copyTpl
	return nil
}

// FindTemplateByID retrieves an InspectionTemplate by ID from memory.
func (r *InspectionTemplateRepo) FindTemplateByID(_ context.Context, id string) (*domain.InspectionTemplate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tpl, ok := r.templates[id]
	if !ok {
		return nil, domain.ErrTemplateNotFound
	}
	copyTpl := cloneTemplate(tpl)
	return &copyTpl, nil
}

// ListTemplates returns all InspectionTemplates in insertion order.
func (r *InspectionTemplateRepo) ListTemplates(_ context.Context) ([]domain.InspectionTemplate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]domain.InspectionTemplate, 0, len(r.tplOrder))
	for _, id := range r.tplOrder {
		out = append(out, cloneTemplate(r.templates[id]))
	}
	return out, nil
}

func cloneTemplate(tpl domain.InspectionTemplate) domain.InspectionTemplate {
	copyTpl := tpl
	copyTpl.Rooms = append([]string(nil), tpl.Rooms...)
	copyTpl.DefaultElements = append([]string(nil), tpl.DefaultElements...)
	return copyTpl
}
