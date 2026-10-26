package memory

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/modules/inspection/domain"
)

// InspectionRepo is a thread-safe in-memory implementation of domain.InspectionRepository for tests.
type InspectionRepo struct {
	mu          sync.RWMutex
	inspections map[string]domain.Inspection
	inspOrder   []string
}

// NewInspectionRepo constructs an empty thread-safe in-memory InspectionRepo.
func NewInspectionRepo() *InspectionRepo {
	return &InspectionRepo{
		inspections: make(map[string]domain.Inspection),
	}
}

// SaveInspection stores an Inspection in memory.
func (r *InspectionRepo) SaveInspection(_ context.Context, insp domain.Inspection) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	copyInsp := cloneInspection(insp)
	if existing, exists := r.inspections[insp.ID]; !exists {
		r.inspOrder = append(r.inspOrder, insp.ID)
	} else if len(copyInsp.Items) == 0 && len(existing.Items) > 0 {
		copyInsp.Items = cloneItems(existing.Items)
	}
	r.inspections[insp.ID] = copyInsp
	return nil
}

// FindInspectionByID retrieves an Inspection by ID from memory.
func (r *InspectionRepo) FindInspectionByID(_ context.Context, id string) (*domain.Inspection, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	insp, ok := r.inspections[id]
	if !ok {
		return nil, domain.ErrInspectionNotFound
	}
	copyInsp := cloneInspection(insp)
	return &copyInsp, nil
}

// ListInspectionsByLease returns all Inspections belonging to leaseID.
func (r *InspectionRepo) ListInspectionsByLease(_ context.Context, leaseID string) ([]domain.Inspection, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []domain.Inspection
	for _, id := range r.inspOrder {
		insp := r.inspections[id]
		if insp.LeaseID == leaseID {
			out = append(out, cloneInspection(insp))
		}
	}
	return out, nil
}

// UpsertInspectionItems merges items by (RoomName, ElementName) into the specified inspection in memory.
func (r *InspectionRepo) UpsertInspectionItems(_ context.Context, inspectionID string, items []domain.InspectionItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	insp, ok := r.inspections[inspectionID]
	if !ok {
		return domain.ErrInspectionNotFound
	}

	indexByKey := make(map[string]int, len(insp.Items))
	for i, existing := range insp.Items {
		indexByKey[itemCompositeKey(existing.RoomName, existing.ElementName)] = i
	}

	for _, item := range items {
		copyItem := item
		copyItem.PhotoR2Keys = append([]string(nil), item.PhotoR2Keys...)
		key := itemCompositeKey(item.RoomName, item.ElementName)
		if idx, exists := indexByKey[key]; exists {
			insp.Items[idx] = copyItem
		} else {
			indexByKey[key] = len(insp.Items)
			insp.Items = append(insp.Items, copyItem)
		}
	}

	insp.UpdatedAt = time.Now().UTC()
	r.inspections[inspectionID] = insp
	return nil
}

func itemCompositeKey(room, element string) string {
	return strings.ToLower(strings.TrimSpace(room)) + "::" + strings.ToLower(strings.TrimSpace(element))
}

func cloneInspection(insp domain.Inspection) domain.Inspection {
	copyInsp := insp
	if insp.EntryInspectionID != nil {
		v := *insp.EntryInspectionID
		copyInsp.EntryInspectionID = &v
	}
	copyInsp.Items = cloneItems(insp.Items)
	return copyInsp
}

func cloneItems(items []domain.InspectionItem) []domain.InspectionItem {
	out := make([]domain.InspectionItem, len(items))
	for i, it := range items {
		copyIt := it
		copyIt.PhotoR2Keys = append([]string(nil), it.PhotoR2Keys...)
		out[i] = copyIt
	}
	return out
}
