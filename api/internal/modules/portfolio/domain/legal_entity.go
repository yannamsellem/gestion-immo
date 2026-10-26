package domain

import (
	"context"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

// EntityType represents the legal and fiscal classification of a portfolio entity.
type EntityType string

const (
	EntityTypeSCIIR              EntityType = "SCI_IR"
	EntityTypeSCIIS              EntityType = "SCI_IS"
	EntityTypeSARLIS             EntityType = "SARL_IS"
	EntityTypeFamilySARL         EntityType = "FAMILY_SARL"
	EntityTypeSAS                EntityType = "SAS"
	EntityTypeSoleProprietorship EntityType = "SOLE_PROPRIETORSHIP"
	EntityTypeJointOwnership     EntityType = "JOINT_OWNERSHIP"
)

// IsValid reports whether the EntityType is a supported portfolio legal entity type.
func (t EntityType) IsValid() bool {
	switch t {
	case EntityTypeSCIIR,
		EntityTypeSCIIS,
		EntityTypeSARLIS,
		EntityTypeFamilySARL,
		EntityTypeSAS,
		EntityTypeSoleProprietorship,
		EntityTypeJointOwnership:
		return true
	default:
		return false
	}
}

// LegalEntity is the aggregate root representing a legal ownership structure
// (SCI IR/IS, SARL IS / Real Estate Dealer, Family SARL, SAS, Sole Proprietorship, Joint Ownership).
type LegalEntity struct {
	ID                string
	Name              string
	EntityType        EntityType
	SIRET             string
	NAFCode           string
	Address           string
	ShareCapitalCents money.AmountCents
	TotalShares       int64
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// LegalEntityRepository defines the persistence contract dedicated to the LegalEntity aggregate.
type LegalEntityRepository interface {
	CreateLegalEntity(ctx context.Context, entity LegalEntity) error
	FindLegalEntityByID(ctx context.Context, id string) (*LegalEntity, error)
	ListLegalEntities(ctx context.Context) ([]LegalEntity, error)
}
