package domain

import (
	"context"
	"time"
)

// Tenant is the aggregate representing an active or archived tenant (locataire présent ou archivé).
type Tenant struct {
	ID         string
	FullName   string
	Email      string
	Phone      string
	Address    string
	Archived   bool
	ArchivedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// TenantRepository defines the persistence contract dedicated to the Tenant aggregate.
type TenantRepository interface {
	CreateTenant(ctx context.Context, tenant Tenant) error
	FindTenantByID(ctx context.Context, id string) (*Tenant, error)
	ArchiveTenant(ctx context.Context, id string, archivedAt time.Time) error
	ListTenants(ctx context.Context, includeArchived bool) ([]Tenant, error)
}
