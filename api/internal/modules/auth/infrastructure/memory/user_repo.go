package memory

import (
	"context"
	"strings"
	"sync"

	"github.com/yannamsellem/gestion-immo/internal/modules/auth/domain"
)

// UserRepo is a thread-safe in-memory implementation of domain.UserRepository for tests.
type UserRepo struct {
	mu           sync.RWMutex
	usersByID    map[string]domain.User
	usersByEmail map[string]string
}

// NewUserRepo constructs an empty thread-safe in-memory UserRepo.
func NewUserRepo() *UserRepo {
	return &UserRepo{
		usersByID:    make(map[string]domain.User),
		usersByEmail: make(map[string]string),
	}
}

// CreateUser stores a new user in memory.
func (r *UserRepo) CreateUser(_ context.Context, user domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	emailKey := strings.ToLower(strings.TrimSpace(user.Email))
	if _, exists := r.usersByEmail[emailKey]; exists {
		return domain.ErrEmailAlreadyExists
	}
	user.Email = emailKey
	r.usersByID[user.ID] = user
	r.usersByEmail[emailKey] = user.ID
	return nil
}

// UpdateUser updates an existing user in memory.
func (r *UserRepo) UpdateUser(_ context.Context, user domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.usersByID[user.ID]; !exists {
		return domain.ErrUserNotFound
	}
	emailKey := strings.ToLower(strings.TrimSpace(user.Email))
	user.Email = emailKey
	r.usersByID[user.ID] = user
	r.usersByEmail[emailKey] = user.ID
	return nil
}

// FindUserByID retrieves a user by ID from memory.
func (r *UserRepo) FindUserByID(_ context.Context, id string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	u, ok := r.usersByID[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	copyUser := u
	return &copyUser, nil
}

// FindUserByEmail retrieves a user by email from memory.
func (r *UserRepo) FindUserByEmail(_ context.Context, email string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.usersByEmail[strings.ToLower(strings.TrimSpace(email))]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	u := r.usersByID[id]
	return &u, nil
}

// ListUsersByStatus lists all users with the given status from memory.
func (r *UserRepo) ListUsersByStatus(_ context.Context, status domain.UserStatus) ([]domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []domain.User
	for _, u := range r.usersByID {
		if u.Status == status {
			out = append(out, u)
		}
	}
	return out, nil
}
