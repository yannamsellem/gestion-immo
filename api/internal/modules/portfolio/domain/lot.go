package domain

import (
	"context"
	"time"
)

// Lot is the entity/aggregate representing an individual rentable unit
// (apartment, commercial shop, parking space, cellar) belonging to a Property.
type Lot struct {
	ID             string
	PropertyID     string
	Label          string
	LotType        string
	SurfaceM2      int64
	OwnershipShare int64
	Floor          string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// LotRepository defines the persistence contract dedicated to the Lot aggregate.
type LotRepository interface {
	CreateLot(ctx context.Context, lot Lot) error
	ListLotsByProperty(ctx context.Context, propertyID string) ([]Lot, error)
}
