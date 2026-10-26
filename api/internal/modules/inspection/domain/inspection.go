package domain

import (
	"context"
	"time"
)

// InspectionType distinguishes move-in (MOVE_IN) from move-out (MOVE_OUT) property inspections.
type InspectionType string

const (
	InspectionTypeMoveIn  InspectionType = "MOVE_IN"
	InspectionTypeMoveOut InspectionType = "MOVE_OUT"
)

// IsValid reports whether t is a supported InspectionType.
func (t InspectionType) IsValid() bool {
	switch t {
	case InspectionTypeMoveIn, InspectionTypeMoveOut:
		return true
	default:
		return false
	}
}

// ConditionState represents the standardized condition rating of a room element during an inspection.
type ConditionState string

const (
	ConditionNew          ConditionState = "NEW"
	ConditionGood         ConditionState = "GOOD"
	ConditionFairWear     ConditionState = "FAIR_WEAR"
	ConditionPoor         ConditionState = "POOR"
	ConditionOutOfService ConditionState = "OUT_OF_SERVICE"
)

// IsValid reports whether c is a valid ConditionState.
func (c ConditionState) IsValid() bool {
	switch c {
	case ConditionNew, ConditionGood, ConditionFairWear, ConditionPoor, ConditionOutOfService:
		return true
	default:
		return false
	}
}

// SeverityScore returns a monotonic numeric rank (1 = best, 5 = worst) used to detect exit degradations.
func (c ConditionState) SeverityScore() int {
	switch c {
	case ConditionNew:
		return 1
	case ConditionGood:
		return 2
	case ConditionFairWear:
		return 3
	case ConditionPoor:
		return 4
	case ConditionOutOfService:
		return 5
	default:
		return 0
	}
}

// IsDegraded reports whether an element's condition deteriorated between entry and exit inspections.
func IsDegraded(entry, exit ConditionState) bool {
	if !entry.IsValid() || !exit.IsValid() {
		return false
	}
	return exit.SeverityScore() > entry.SeverityScore()
}

// InspectionItem captures the observed condition, notes, and R2 photo keys of a single room element.
// It is a child entity owned by the Inspection aggregate root.
type InspectionItem struct {
	ID           string
	InspectionID string
	RoomName     string
	ElementName  string
	Condition    ConditionState
	Observation  string
	PhotoR2Keys  []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Inspection represents the Aggregate Root for a structured Move-In (MOVE_IN) or Move-Out (MOVE_OUT)
// property inspection along with its child InspectionItems.
// EntryInspectionID is optional (*string): legacy leases that only have a scanned PDF entry inspection
// can still create a standalone MOVE_OUT inspection with EntryInspectionID == nil.
type Inspection struct {
	ID                string
	LeaseID           string
	TemplateID        string
	Type              InspectionType
	EntryInspectionID *string
	Items             []InspectionItem
	ReportPDFR2Key    string
	InspectionDate    time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// InspectionRepository defines the persistence contract for the Inspection aggregate root (including its InspectionItems).
type InspectionRepository interface {
	SaveInspection(ctx context.Context, insp Inspection) error
	FindInspectionByID(ctx context.Context, id string) (*Inspection, error)
	ListInspectionsByLease(ctx context.Context, leaseID string) ([]Inspection, error)
	UpsertInspectionItems(ctx context.Context, inspectionID string, items []InspectionItem) error
}
