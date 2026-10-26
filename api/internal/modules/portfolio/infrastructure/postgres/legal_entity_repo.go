package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yannamsellem/gestion-immo/internal/modules/portfolio/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
	"github.com/yannamsellem/gestion-immo/internal/platform/postgres"
)

// LegalEntityRepo implements domain.LegalEntityRepository against PostgreSQL.
type LegalEntityRepo struct {
	db postgres.DBTX
}

// NewLegalEntityRepo constructs a PostgreSQL repository dedicated to LegalEntity.
func NewLegalEntityRepo(db postgres.DBTX) *LegalEntityRepo {
	return &LegalEntityRepo{db: db}
}

// CreateLegalEntity inserts a new legal entity row into PostgreSQL.
func (r *LegalEntityRepo) CreateLegalEntity(ctx context.Context, entity domain.LegalEntity) error {
	const query = `
		INSERT INTO legal_entities (
			id, name, entity_type, siret, naf_code, address,
			share_capital_cents, total_shares, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := postgres.Conn(ctx, r.db).Exec(ctx, query,
		entity.ID,
		entity.Name,
		string(entity.EntityType),
		entity.SIRET,
		entity.NAFCode,
		entity.Address,
		int64(entity.ShareCapitalCents),
		entity.TotalShares,
		entity.CreatedAt,
		entity.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrDuplicateSIRET
		}
		return fmt.Errorf("portfolio/postgres/legal_entity_repo: create legal entity: %w", err)
	}
	return nil
}

// FindLegalEntityByID retrieves a legal entity by primary key.
func (r *LegalEntityRepo) FindLegalEntityByID(ctx context.Context, id string) (*domain.LegalEntity, error) {
	const query = `
		SELECT id, name, entity_type, siret, naf_code, address,
		       share_capital_cents, total_shares, created_at, updated_at
		FROM legal_entities
		WHERE id = $1
	`
	row := postgres.Conn(ctx, r.db).QueryRow(ctx, query, id)
	return scanLegalEntity(row)
}

// ListLegalEntities lists all legal entities ordered by creation date.
func (r *LegalEntityRepo) ListLegalEntities(ctx context.Context) ([]domain.LegalEntity, error) {
	const query = `
		SELECT id, name, entity_type, siret, naf_code, address,
		       share_capital_cents, total_shares, created_at, updated_at
		FROM legal_entities
		ORDER BY created_at ASC
	`
	rows, err := postgres.Conn(ctx, r.db).Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("portfolio/postgres/legal_entity_repo: list legal entities: %w", err)
	}
	defer rows.Close()

	var entities []domain.LegalEntity
	for rows.Next() {
		ent, err := scanLegalEntity(rows)
		if err != nil {
			return nil, err
		}
		entities = append(entities, *ent)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("portfolio/postgres/legal_entity_repo: iterate legal entities: %w", err)
	}
	return entities, nil
}

func scanLegalEntity(row pgx.Row) (*domain.LegalEntity, error) {
	var e domain.LegalEntity
	var typeStr string
	var capCents int64
	err := row.Scan(
		&e.ID,
		&e.Name,
		&typeStr,
		&e.SIRET,
		&e.NAFCode,
		&e.Address,
		&capCents,
		&e.TotalShares,
		&e.CreatedAt,
		&e.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrLegalEntityNotFound
		}
		return nil, fmt.Errorf("portfolio/postgres/legal_entity_repo: scan legal entity: %w", err)
	}
	e.EntityType = domain.EntityType(typeStr)
	e.ShareCapitalCents = money.AmountCents(capCents)
	return &e, nil
}
