package memory

import (
	"context"
	"sync"

	"github.com/yannamsellem/gestion-immo/internal/modules/document/domain"
)

// StoredFileRepo is a thread-safe in-memory implementation of domain.StoredFileRepository for tests.
type StoredFileRepo struct {
	mu          sync.RWMutex
	storedFiles map[string]domain.StoredFile
	fileOrder   []string
}

// NewStoredFileRepo constructs an empty thread-safe in-memory StoredFileRepo.
func NewStoredFileRepo() *StoredFileRepo {
	return &StoredFileRepo{
		storedFiles: make(map[string]domain.StoredFile),
	}
}

// SaveStoredFile stores a StoredFile metadata record in memory.
func (r *StoredFileRepo) SaveStoredFile(_ context.Context, file domain.StoredFile) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.storedFiles[file.ID]; !exists {
		r.fileOrder = append(r.fileOrder, file.ID)
	}
	r.storedFiles[file.ID] = file
	return nil
}

// FindStoredFileByID retrieves a StoredFile by ID from memory.
func (r *StoredFileRepo) FindStoredFileByID(_ context.Context, id string) (*domain.StoredFile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	f, ok := r.storedFiles[id]
	if !ok {
		return nil, domain.ErrStoredFileNotFound
	}
	copyF := f
	return &copyF, nil
}

// ListStoredFilesByLease returns all StoredFiles belonging to leaseID.
func (r *StoredFileRepo) ListStoredFilesByLease(_ context.Context, leaseID string) ([]domain.StoredFile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []domain.StoredFile
	for _, id := range r.fileOrder {
		f := r.storedFiles[id]
		if f.LeaseID == leaseID {
			out = append(out, f)
		}
	}
	return out, nil
}

// ListStoredFilesByEntity returns all StoredFiles belonging to entityID.
func (r *StoredFileRepo) ListStoredFilesByEntity(_ context.Context, entityID string) ([]domain.StoredFile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []domain.StoredFile
	for _, id := range r.fileOrder {
		f := r.storedFiles[id]
		if f.LegalEntityID == entityID {
			out = append(out, f)
		}
	}
	return out, nil
}
