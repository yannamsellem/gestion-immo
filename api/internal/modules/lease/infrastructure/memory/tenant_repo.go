package memory

import (
	"context"
	"sync"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/modules/lease/domain"
)

// TenantRepo is a thread-safe in-memory implementation of domain.TenantRepository for tests.
type TenantRepo struct {
	mu          sync.RWMutex
	tenants     map[string]domain.Tenant
	tenantOrder []string
}

// NewTenantRepo constructs an empty thread-safe in-memory TenantRepo.
func NewTenantRepo() *TenantRepo {
	return &TenantRepo{
		tenants: make(map[string]domain.Tenant),
	}
}

// CreateTenant stores a new Tenant in memory.
func (r *TenantRepo) CreateTenant(_ context.Context, tenant domain.Tenant) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tenants[tenant.ID] = tenant
	r.tenantOrder = append(r.tenantOrder, tenant.ID)
	return nil
}

// FindTenantByID retrieves a Tenant by ID from memory.
func (r *TenantRepo) FindTenantByID(_ context.Context, id string) (*domain.Tenant, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.tenants[id]
	if !ok {
		return nil, domain.ErrTenantNotFound
	}
	copyT := t
	return &copyT, nil
}

// ArchiveTenant marks a Tenant as archived in memory.
func (r *TenantRepo) ArchiveTenant(_ context.Context, id string, archivedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	t, ok := r.tenants[id]
	if !ok {
		return domain.ErrTenantNotFound
	}
	ts := archivedAt
	t.Archived = true
	t.ArchivedAt = &ts
	t.UpdatedAt = archivedAt
	r.tenants[id] = t
	return nil
}

// ListTenants lists tenants in insertion order, optionally filtering out archived ones.
func (r *TenantRepo) ListTenants(_ context.Context, includeArchived bool) ([]domain.Tenant, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]domain.Tenant, 0, len(r.tenantOrder))
	for _, id := range r.tenantOrder {
		t := r.tenants[id]
		if !includeArchived && t.Archived {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}
