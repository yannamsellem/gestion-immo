package domain

import (
	"context"
	"time"
)

// StoredFile represents the Aggregate Root for metadata of a document, attachment, or generated archive stored on Cloudflare R2.
type StoredFile struct {
	ID            string
	LegalEntityID string
	LeaseID       string
	Category      string
	FileName      string
	ContentType   string
	R2Key         string
	SizeBytes     int64
	CreatedAt     time.Time
}

// StoredFileRepository defines the persistence contract for the StoredFile aggregate root.
type StoredFileRepository interface {
	SaveStoredFile(ctx context.Context, file StoredFile) error
	FindStoredFileByID(ctx context.Context, id string) (*StoredFile, error)
	ListStoredFilesByLease(ctx context.Context, leaseID string) ([]StoredFile, error)
	ListStoredFilesByEntity(ctx context.Context, entityID string) ([]StoredFile, error)
}
