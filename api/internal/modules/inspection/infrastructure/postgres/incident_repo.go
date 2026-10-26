package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/yannamsellem/gestion-immo/internal/modules/inspection/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
	"github.com/yannamsellem/gestion-immo/internal/platform/postgres"
)

// IncidentRepo implements domain.IncidentRepository using postgres.DBTX.
type IncidentRepo struct {
	db postgres.DBTX
}

// NewIncidentRepo constructs a new PostgreSQL repository for Incident aggregates.
func NewIncidentRepo(db postgres.DBTX) *IncidentRepo {
	return &IncidentRepo{db: db}
}

// SaveIncident inserts or updates an Incident in PostgreSQL.
func (r *IncidentRepo) SaveIncident(ctx context.Context, inc domain.Incident) error {
	const query = `
		INSERT INTO incidents (
			id, property_id, lot_id, lease_id, title, status,
			insurance_claim_number, estimated_cost_cents, observations, photo_r2_keys,
			occurred_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10,
			$11, $12, $13
		)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			insurance_claim_number = EXCLUDED.insurance_claim_number,
			estimated_cost_cents = EXCLUDED.estimated_cost_cents,
			observations = EXCLUDED.observations,
			photo_r2_keys = EXCLUDED.photo_r2_keys,
			updated_at = EXCLUDED.updated_at
	`
	obs := inc.Observations
	if obs == nil {
		obs = []string{}
	}
	photos := inc.PhotoR2Keys
	if photos == nil {
		photos = []string{}
	}
	_, err := postgres.Conn(ctx, r.db).Exec(ctx, query,
		inc.ID,
		inc.PropertyID,
		inc.LotID,
		inc.LeaseID,
		inc.Title,
		string(inc.Status),
		inc.InsuranceClaimNumber,
		int64(inc.EstimatedCostCents),
		obs,
		photos,
		inc.OccurredAt,
		inc.CreatedAt,
		inc.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inspection/postgres/incident_repo: save incident %s: %w", inc.ID, err)
	}
	return nil
}

// FindIncidentByID retrieves an Incident by ID.
func (r *IncidentRepo) FindIncidentByID(ctx context.Context, id string) (*domain.Incident, error) {
	const query = `
		SELECT id, property_id, lot_id, lease_id, title, status,
		       insurance_claim_number, estimated_cost_cents, observations, photo_r2_keys,
		       occurred_at, created_at, updated_at
		FROM incidents
		WHERE id = $1
	`
	var inc domain.Incident
	var status string
	var costCents int64
	err := postgres.Conn(ctx, r.db).QueryRow(ctx, query, id).Scan(
		&inc.ID,
		&inc.PropertyID,
		&inc.LotID,
		&inc.LeaseID,
		&inc.Title,
		&status,
		&inc.InsuranceClaimNumber,
		&costCents,
		&inc.Observations,
		&inc.PhotoR2Keys,
		&inc.OccurredAt,
		&inc.CreatedAt,
		&inc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrIncidentNotFound
		}
		return nil, fmt.Errorf("inspection/postgres/incident_repo: find incident by id: %w", err)
	}
	inc.Status = domain.IncidentStatus(status)
	inc.EstimatedCostCents = money.AmountCents(costCents)
	return &inc, nil
}

// ListIncidents lists Incidents filtered optionally by propertyID and/or leaseID.
func (r *IncidentRepo) ListIncidents(ctx context.Context, propertyID, leaseID string) ([]domain.Incident, error) {
	const query = `
		SELECT id, property_id, lot_id, lease_id, title, status,
		       insurance_claim_number, estimated_cost_cents, observations, photo_r2_keys,
		       occurred_at, created_at, updated_at
		FROM incidents
		WHERE ($1 = '' OR property_id = $1)
		  AND ($2 = '' OR lease_id = $2)
		ORDER BY occurred_at DESC, created_at DESC
	`
	rows, err := postgres.Conn(ctx, r.db).Query(ctx, query, propertyID, leaseID)
	if err != nil {
		return nil, fmt.Errorf("inspection/postgres/incident_repo: list incidents: %w", err)
	}
	defer rows.Close()

	var out []domain.Incident
	for rows.Next() {
		var inc domain.Incident
		var status string
		var costCents int64
		if err := rows.Scan(
			&inc.ID,
			&inc.PropertyID,
			&inc.LotID,
			&inc.LeaseID,
			&inc.Title,
			&status,
			&inc.InsuranceClaimNumber,
			&costCents,
			&inc.Observations,
			&inc.PhotoR2Keys,
			&inc.OccurredAt,
			&inc.CreatedAt,
			&inc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("inspection/postgres/incident_repo: scan incident: %w", err)
		}
		inc.Status = domain.IncidentStatus(status)
		inc.EstimatedCostCents = money.AmountCents(costCents)
		out = append(out, inc)
	}
	return out, rows.Err()
}
