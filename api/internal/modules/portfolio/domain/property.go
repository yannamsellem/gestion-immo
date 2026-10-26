package domain

import (
	"context"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

// AccountingCategory represents the PCG balance sheet classification of a property asset.
type AccountingCategory string

const (
	AccountingCategoryFixedAsset   AccountingCategory = "FIXED_ASSET"
	AccountingCategoryMerchandise  AccountingCategory = "MERCHANDISE"
	AccountingCategoryFinishedGood AccountingCategory = "FINISHED_GOOD"
)

// IsValid reports whether the AccountingCategory is a supported PCG classification.
func (c AccountingCategory) IsValid() bool {
	switch c {
	case AccountingCategoryFixedAsset,
		AccountingCategoryMerchandise,
		AccountingCategoryFinishedGood:
		return true
	default:
		return false
	}
}

// Property is the aggregate representing a real estate asset owned by a LegalEntity, classified either as
// a long-term fixed asset (FIXED_ASSET) or real estate dealer inventory (MERCHANDISE / FINISHED_GOOD)
// with capitalized renovation work stock.
type Property struct {
	ID                  string
	LegalEntityID       string
	Label               string
	PropertyType        string
	AccountingCategory  AccountingCategory
	Address             string
	AcquisitionCents    money.AmountCents
	WorkStockCents      money.AmountCents
	TotalBookValueCents money.AmountCents
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// ComputeTotalBookValue returns AcquisitionCents + WorkStockCents.
func (p Property) ComputeTotalBookValue() money.AmountCents {
	return p.AcquisitionCents.Add(p.WorkStockCents)
}

// PropertyRepository defines the persistence contract dedicated to the Property aggregate.
type PropertyRepository interface {
	CreateProperty(ctx context.Context, property Property) error
	FindPropertyByID(ctx context.Context, id string) (*Property, error)
	ListPropertiesByEntity(ctx context.Context, entityID string) ([]Property, error)
}
