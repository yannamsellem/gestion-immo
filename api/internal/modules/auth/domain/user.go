package domain

import (
	"context"
	"time"
)

// UserRole represents the authorization role of an authenticated back-office user.
type UserRole string

const (
	RoleSuperAdmin UserRole = "SUPER_ADMIN"
	RoleManager    UserRole = "MANAGER"
)

// UserStatus represents the approval lifecycle state of a user account.
type UserStatus string

const (
	StatusPendingApproval UserStatus = "PENDING_APPROVAL"
	StatusApproved        UserStatus = "APPROVED"
	StatusRejected        UserStatus = "REJECTED"
)

// User is the aggregate root representing a back-office operator or super administrator.
type User struct {
	ID           string
	FullName     string
	Email        string
	PasswordHash string
	Role         UserRole
	Status       UserStatus
	ApprovedBy   *string
	ApprovedAt   *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// IsApproved reports whether the user account has been approved by a SUPER_ADMIN.
func (u User) IsApproved() bool {
	return u.Status == StatusApproved
}

// IsSuperAdmin reports whether the user holds the SUPER_ADMIN role and is approved.
func (u User) IsSuperAdmin() bool {
	return u.Role == RoleSuperAdmin && u.Status == StatusApproved
}

// UserRepository defines the persistence contract dedicated to the User aggregate.
type UserRepository interface {
	CreateUser(ctx context.Context, user User) error
	UpdateUser(ctx context.Context, user User) error
	FindUserByID(ctx context.Context, id string) (*User, error)
	FindUserByEmail(ctx context.Context, email string) (*User, error)
	ListUsersByStatus(ctx context.Context, status UserStatus) ([]User, error)
}
