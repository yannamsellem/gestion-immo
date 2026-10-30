package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/yannamsellem/gestion-immo/internal/modules/auth/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/id"
	platformpg "github.com/yannamsellem/gestion-immo/internal/platform/postgres"
)

// UserRepo implements domain.UserRepository against PostgreSQL.
type UserRepo struct {
	db platformpg.DBTX
}

// NewUserRepo constructs a PostgreSQL repository dedicated to the User aggregate.
func NewUserRepo(db platformpg.DBTX) *UserRepo {
	return &UserRepo{db: db}
}

// CreateUser inserts a new user row into PostgreSQL.
func (r *UserRepo) CreateUser(ctx context.Context, user domain.User) error {
	const query = `
		INSERT INTO users (id, full_name, email, password_hash, role, status, approved_by, approved_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := platformpg.Conn(ctx, r.db).Exec(ctx, query,
		user.ID,
		user.FullName,
		strings.ToLower(strings.TrimSpace(user.Email)),
		user.PasswordHash,
		string(user.Role),
		string(user.Status),
		user.ApprovedBy,
		user.ApprovedAt,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrEmailAlreadyExists
		}
		if platformpg.IsForeignKeyViolation(err) {
			return fmt.Errorf("%w: referenced user does not exist", domain.ErrValidation)
		}
		return fmt.Errorf("user_postgres_repo: create user: %w", err)
	}
	return nil
}

// UpdateUser updates an existing user row in PostgreSQL.
func (r *UserRepo) UpdateUser(ctx context.Context, user domain.User) error {
	if !id.IsValidUUID(user.ID) {
		return domain.ErrUserNotFound
	}
	const query = `
		UPDATE users
		SET full_name = $2,
		    email = $3,
		    password_hash = $4,
		    role = $5,
		    status = $6,
		    approved_by = $7,
		    approved_at = $8,
		    updated_at = $9
		WHERE id = $1
	`
	tag, err := platformpg.Conn(ctx, r.db).Exec(ctx, query,
		user.ID,
		user.FullName,
		strings.ToLower(strings.TrimSpace(user.Email)),
		user.PasswordHash,
		string(user.Role),
		string(user.Status),
		user.ApprovedBy,
		user.ApprovedAt,
		user.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrEmailAlreadyExists
		}
		if platformpg.IsForeignKeyViolation(err) {
			return fmt.Errorf("%w: referenced user does not exist", domain.ErrValidation)
		}
		return fmt.Errorf("user_postgres_repo: update user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

// FindUserByID retrieves a user by primary key.
func (r *UserRepo) FindUserByID(ctx context.Context, userID string) (*domain.User, error) {
	if !id.IsValidUUID(userID) {
		return nil, domain.ErrUserNotFound
	}
	const query = `
		SELECT id, full_name, email, password_hash, role, status, approved_by, approved_at, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	row := platformpg.Conn(ctx, r.db).QueryRow(ctx, query, userID)
	return scanUser(row)
}

// FindUserByEmail retrieves a user by case-insensitive email address.
func (r *UserRepo) FindUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	const query = `
		SELECT id, full_name, email, password_hash, role, status, approved_by, approved_at, created_at, updated_at
		FROM users
		WHERE email = LOWER($1)
	`
	row := platformpg.Conn(ctx, r.db).QueryRow(ctx, query, strings.TrimSpace(email))
	return scanUser(row)
}

// ListUsersByStatus lists all users matching the given approval status ordered by creation time.
func (r *UserRepo) ListUsersByStatus(ctx context.Context, status domain.UserStatus) ([]domain.User, error) {
	const query = `
		SELECT id, full_name, email, password_hash, role, status, approved_by, approved_at, created_at, updated_at
		FROM users
		WHERE status = $1
		ORDER BY created_at ASC
	`
	rows, err := platformpg.Conn(ctx, r.db).Query(ctx, query, string(status))
	if err != nil {
		return nil, fmt.Errorf("user_postgres_repo: list users by status: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("user_postgres_repo: iterate users: %w", err)
	}
	return users, nil
}

func scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	var roleStr, statusStr string
	err := row.Scan(
		&u.ID,
		&u.FullName,
		&u.Email,
		&u.PasswordHash,
		&roleStr,
		&statusStr,
		&u.ApprovedBy,
		&u.ApprovedAt,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("user_postgres_repo: scan user: %w", err)
	}
	u.Role = domain.UserRole(roleStr)
	u.Status = domain.UserStatus(statusStr)
	return &u, nil
}
