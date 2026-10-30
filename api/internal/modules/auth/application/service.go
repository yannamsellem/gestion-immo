package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/modules/auth/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/id"
	"github.com/yannamsellem/gestion-immo/internal/platform/secret"
)

const defaultRefreshTTL = 30 * 24 * time.Hour

// SignupCommand carries the input for registering a new back-office account.
// New signups always start in PENDING_APPROVAL with role MANAGER.
type SignupCommand struct {
	FullName string
	Email    string
	Password secret.String
}

// LogValue implements slog.LogValuer to guarantee passwords are never logged.
func (c SignupCommand) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("full_name", c.FullName),
		slog.String("email", c.Email),
		slog.Any("password", c.Password),
	)
}

// LoginCommand carries the credentials for authenticating an approved user.
type LoginCommand struct {
	Email    string
	Password secret.String
}

// LogValue implements slog.LogValuer to guarantee passwords are never logged.
func (c LoginCommand) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("email", c.Email),
		slog.Any("password", c.Password),
	)
}

// UserDTO is an immutable projection of a domain.User safe for external transport.
type UserDTO struct {
	ID         string            `json:"id" format:"uuid"`
	FullName   string            `json:"full_name"`
	Email      string            `json:"email"`
	Role       domain.UserRole   `json:"role"`
	Status     domain.UserStatus `json:"status"`
	ApprovedBy *string           `json:"approved_by,omitempty" format:"uuid"`
	ApprovedAt *time.Time        `json:"approved_at,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
}

// AuthTokensDTO holds the issued JWT access token and rotating refresh token for an approved user.
type AuthTokensDTO struct {
	AccessToken  secret.String
	RefreshToken secret.String
	TokenType    string
	User         UserDTO
}

// LogValue implements slog.LogValuer to guarantee JWTs and refresh tokens are never logged.
func (t AuthTokensDTO) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("access_token", t.AccessToken),
		slog.Any("refresh_token", t.RefreshToken),
		slog.String("token_type", t.TokenType),
		slog.String("user_id", t.User.ID),
	)
}

// Service orchestrates authentication, SUPER_ADMIN signup approval workflows, and MCP token lifecycle.
type Service struct {
	userRepo    domain.UserRepository
	refreshRepo domain.RefreshTokenRepository
	mcpRepo     domain.MCPTokenRepository
	uow         domain.UnitOfWork
	jwtSvc      domain.AccessTokenManager
	hasher      domain.SecretHasher
	dummyHash   string
}

// NewService constructs an auth application Service with constructor-injected per-aggregate repositories and UnitOfWork.
func NewService(
	userRepo domain.UserRepository,
	refreshRepo domain.RefreshTokenRepository,
	mcpRepo domain.MCPTokenRepository,
	uow domain.UnitOfWork,
	jwtSvc domain.AccessTokenManager,
	hasher domain.SecretHasher,
) *Service {
	var dummyHash string
	if hasher != nil {
		dummyHash, _ = hasher.HashPassword(secret.New("dummy-constant-time-password"))
	}
	return &Service{
		userRepo:    userRepo,
		refreshRepo: refreshRepo,
		mcpRepo:     mcpRepo,
		uow:         uow,
		jwtSvc:      jwtSvc,
		hasher:      hasher,
		dummyHash:   dummyHash,
	}
}

// BootstrapSuperAdmin creates or ensures the initial SUPER_ADMIN account in APPROVED status.
func (s *Service) BootstrapSuperAdmin(ctx context.Context, email string, password secret.String) (UserDTO, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	if normalizedEmail == "" || strings.TrimSpace(password.Value()) == "" {
		return UserDTO{}, fmt.Errorf("%w: email and password are required", domain.ErrValidation)
	}
	if len(password.Value()) < 8 || len(password.Value()) > domain.MaxPasswordBytes {
		return UserDTO{}, fmt.Errorf("%w: password must be between 8 and %d bytes", domain.ErrValidation, domain.MaxPasswordBytes)
	}

	passHash, err := s.hasher.HashPassword(password)
	if err != nil {
		return UserDTO{}, err
	}

	now := time.Now().UTC()
	existing, err := s.userRepo.FindUserByEmail(ctx, normalizedEmail)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return UserDTO{}, err
	}
	if existing != nil {
		existing.Role = domain.RoleSuperAdmin
		existing.Status = domain.StatusApproved
		existing.PasswordHash = passHash
		existing.ApprovedAt = &now
		existing.UpdatedAt = now
		if err := s.userRepo.UpdateUser(ctx, *existing); err != nil {
			return UserDTO{}, err
		}
		return toUserDTO(*existing), nil
	}

	userID, err := id.NewV7()
	if err != nil {
		return UserDTO{}, err
	}
	user := domain.User{
		ID:           userID,
		FullName:     "Super Administrateur",
		Email:        normalizedEmail,
		PasswordHash: passHash,
		Role:         domain.RoleSuperAdmin,
		Status:       domain.StatusApproved,
		ApprovedAt:   &now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return UserDTO{}, err
	}
	return toUserDTO(user), nil
}

// Signup registers a new user account in PENDING_APPROVAL status with role MANAGER.
// No tokens are issued until a SUPER_ADMIN explicitly approves the account.
func (s *Service) Signup(ctx context.Context, cmd SignupCommand) (UserDTO, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(cmd.Email))
	fullName := strings.TrimSpace(cmd.FullName)
	if normalizedEmail == "" || strings.TrimSpace(cmd.Password.Value()) == "" {
		return UserDTO{}, fmt.Errorf("%w: email and password are required", domain.ErrValidation)
	}
	if len(cmd.Password.Value()) < 8 || len(cmd.Password.Value()) > domain.MaxPasswordBytes {
		return UserDTO{}, fmt.Errorf("%w: password must be between 8 and %d bytes", domain.ErrValidation, domain.MaxPasswordBytes)
	}
	if fullName == "" {
		fullName = normalizedEmail
	}

	existing, err := s.userRepo.FindUserByEmail(ctx, normalizedEmail)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return UserDTO{}, err
	}
	if existing != nil {
		return UserDTO{}, domain.ErrEmailAlreadyExists
	}

	passHash, err := s.hasher.HashPassword(cmd.Password)
	if err != nil {
		return UserDTO{}, err
	}

	userID, err := id.NewV7()
	if err != nil {
		return UserDTO{}, err
	}
	now := time.Now().UTC()
	user := domain.User{
		ID:           userID,
		FullName:     fullName,
		Email:        normalizedEmail,
		PasswordHash: passHash,
		Role:         domain.RoleManager,
		Status:       domain.StatusPendingApproval,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return UserDTO{}, err
	}
	return toUserDTO(user), nil
}

// ApproveUser transitions a user account to APPROVED status. Only an approved SUPER_ADMIN may perform this action.
func (s *Service) ApproveUser(ctx context.Context, actorUserID string, targetUserID string) (UserDTO, error) {
	actor, err := s.requireSuperAdmin(ctx, actorUserID)
	if err != nil {
		return UserDTO{}, err
	}

	trimmedTargetID := strings.TrimSpace(targetUserID)
	if !id.IsValidUUID(trimmedTargetID) {
		return UserDTO{}, fmt.Errorf("%w: invalid user_id", domain.ErrValidation)
	}

	target, err := s.userRepo.FindUserByID(ctx, trimmedTargetID)
	if err != nil {
		return UserDTO{}, err
	}

	now := time.Now().UTC()
	target.Status = domain.StatusApproved
	target.ApprovedBy = &actor.ID
	target.ApprovedAt = &now
	target.UpdatedAt = now

	if err := s.userRepo.UpdateUser(ctx, *target); err != nil {
		return UserDTO{}, err
	}
	return toUserDTO(*target), nil
}

// RejectUser transitions a user account to REJECTED status and revokes all active refresh tokens
// atomically via the Unit of Work pattern. Only an approved SUPER_ADMIN may perform this action.
func (s *Service) RejectUser(ctx context.Context, actorUserID string, targetUserID string) (UserDTO, error) {
	if _, err := s.requireSuperAdmin(ctx, actorUserID); err != nil {
		return UserDTO{}, err
	}

	trimmedTargetID := strings.TrimSpace(targetUserID)
	if !id.IsValidUUID(trimmedTargetID) {
		return UserDTO{}, fmt.Errorf("%w: invalid user_id", domain.ErrValidation)
	}

	target, err := s.userRepo.FindUserByID(ctx, trimmedTargetID)
	if err != nil {
		return UserDTO{}, err
	}

	now := time.Now().UTC()
	target.Status = domain.StatusRejected
	target.UpdatedAt = now

	if err := s.uow.WithinTransaction(ctx, func(txCtx context.Context, repos domain.Repositories) error {
		if err := repos.Users.UpdateUser(txCtx, *target); err != nil {
			return err
		}
		if err := repos.RefreshTokens.RevokeAllUserRefreshTokens(txCtx, target.ID, now); err != nil {
			return err
		}
		return repos.MCPTokens.RevokeAllUserMCPTokens(txCtx, target.ID, now)
	}); err != nil {
		return UserDTO{}, err
	}

	return toUserDTO(*target), nil
}

// ListPendingUsers returns all accounts currently awaiting SUPER_ADMIN approval.
func (s *Service) ListPendingUsers(ctx context.Context, actorUserID string) ([]UserDTO, error) {
	if _, err := s.requireSuperAdmin(ctx, actorUserID); err != nil {
		return nil, err
	}

	users, err := s.userRepo.ListUsersByStatus(ctx, domain.StatusPendingApproval)
	if err != nil {
		return nil, err
	}

	dtos := make([]UserDTO, 0, len(users))
	for _, u := range users {
		dtos = append(dtos, toUserDTO(u))
	}
	return dtos, nil
}

// Login authenticates a user by email and password, strictly enforcing the SUPER_ADMIN approval gate.
func (s *Service) Login(ctx context.Context, cmd LoginCommand) (AuthTokensDTO, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(cmd.Email))
	if normalizedEmail == "" || cmd.Password.IsEmpty() {
		return AuthTokensDTO{}, domain.ErrInvalidCredentials
	}

	user, err := s.userRepo.FindUserByEmail(ctx, normalizedEmail)
	if err != nil {
		_, _ = s.hasher.VerifyPassword(cmd.Password, s.dummyHash)
		return AuthTokensDTO{}, domain.ErrInvalidCredentials
	}

	ok, err := s.hasher.VerifyPassword(cmd.Password, user.PasswordHash)
	if err != nil || !ok {
		return AuthTokensDTO{}, domain.ErrInvalidCredentials
	}

	if !user.IsApproved() {
		return AuthTokensDTO{}, domain.ErrAccountNotApproved
	}

	return s.issueInitialTokens(ctx, *user)
}

// Refresh validates and rotates a refresh token using Refresh Token Rotation with Reuse Detection.
// If an already-rotated or revoked refresh token is presented, the entire token family is immediately revoked.
func (s *Service) Refresh(ctx context.Context, rawRefreshToken secret.String) (AuthTokensDTO, error) {
	if rawRefreshToken.IsEmpty() {
		return AuthTokensDTO{}, domain.ErrInvalidToken
	}

	tokenHash := s.hasher.HashToken(rawRefreshToken)
	rec, err := s.refreshRepo.FindRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return AuthTokensDTO{}, domain.ErrInvalidToken
	}

	now := time.Now().UTC()
	if rec.IsRevoked() {
		// Refresh Token Rotation Reuse Detection: revoke the entire token family immediately
		_ = s.refreshRepo.RevokeRefreshTokenFamily(ctx, rec.FamilyID, now)
		return AuthTokensDTO{}, domain.ErrRefreshTokenReuseDetected
	}
	if rec.IsExpired(now) {
		return AuthTokensDTO{}, domain.ErrInvalidToken
	}

	user, err := s.userRepo.FindUserByID(ctx, rec.UserID)
	if err != nil {
		return AuthTokensDTO{}, domain.ErrInvalidToken
	}

	if !user.IsApproved() {
		_ = s.refreshRepo.RevokeRefreshTokenFamily(ctx, rec.FamilyID, now)
		return AuthTokensDTO{}, domain.ErrAccountNotApproved
	}

	return s.rotateTokens(ctx, *user, *rec, now)
}

// Logout revokes the provided refresh token if present.
func (s *Service) Logout(ctx context.Context, rawRefreshToken secret.String) error {
	if rawRefreshToken.IsEmpty() {
		return nil
	}
	tokenHash := s.hasher.HashToken(rawRefreshToken)
	return s.refreshRepo.RevokeRefreshTokenByHash(ctx, tokenHash, time.Now().UTC())
}

// CreateMCPToken creates a long-lived Bearer token for Antigravity MCP access, storing only its SHA-256 hash.
func (s *Service) CreateMCPToken(ctx context.Context, userID string, label string) (secret.String, error) {
	trimmedUserID := strings.TrimSpace(userID)
	if !id.IsValidUUID(trimmedUserID) {
		return secret.String{}, fmt.Errorf("%w: invalid user_id", domain.ErrValidation)
	}

	user, err := s.userRepo.FindUserByID(ctx, trimmedUserID)
	if err != nil {
		return secret.String{}, err
	}
	if !user.IsApproved() {
		return secret.String{}, domain.ErrAccountNotApproved
	}

	trimmedLabel := strings.TrimSpace(label)
	if trimmedLabel == "" {
		trimmedLabel = "antigravity-mcp"
	}

	rawTokenStr, err := secret.GeneratePrefixedToken("mcp_live")
	if err != nil {
		return secret.String{}, err
	}
	rawSecret := secret.New(rawTokenStr)
	tokenHash := s.hasher.HashToken(rawSecret)

	tokID, err := id.NewV7()
	if err != nil {
		return secret.String{}, err
	}

	record := domain.MCPApiToken{
		ID:        tokID,
		UserID:    user.ID,
		Label:     trimmedLabel,
		TokenHash: tokenHash,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.mcpRepo.SaveMCPToken(ctx, record); err != nil {
		return secret.String{}, err
	}

	return rawSecret, nil
}

// ValidateMCPToken validates a raw MCP Bearer token and ensures the owning user is APPROVED.
func (s *Service) ValidateMCPToken(ctx context.Context, rawBearer string) (*domain.User, error) {
	trimmed := strings.TrimSpace(rawBearer)
	if trimmed == "" {
		return nil, domain.ErrInvalidToken
	}

	tokenHash := s.hasher.HashToken(secret.New(trimmed))
	tok, err := s.mcpRepo.FindMCPTokenByHash(ctx, tokenHash)
	if err != nil || !tok.IsActive() {
		return nil, domain.ErrInvalidToken
	}

	user, err := s.userRepo.FindUserByID(ctx, tok.UserID)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}
	if !user.IsApproved() {
		return nil, domain.ErrAccountNotApproved
	}

	return user, nil
}

// FindUserByEmail returns the UserDTO for a given email address.
func (s *Service) FindUserByEmail(ctx context.Context, email string) (UserDTO, error) {
	u, err := s.userRepo.FindUserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return UserDTO{}, err
	}
	return toUserDTO(*u), nil
}

// RequireApprovedUser loads a user by ID and returns domain.ErrAccountNotApproved if the account is not APPROVED.
func (s *Service) RequireApprovedUser(ctx context.Context, userID string) (*domain.User, error) {
	user, err := s.userRepo.FindUserByID(ctx, strings.TrimSpace(userID))
	if err != nil {
		return nil, err
	}
	if !user.IsApproved() {
		return nil, domain.ErrAccountNotApproved
	}
	return user, nil
}

// ResolveSuperAdminID locates an approved SUPER_ADMIN account by email (or falls back to the first approved SUPER_ADMIN).
func (s *Service) ResolveSuperAdminID(ctx context.Context, actorEmail string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(actorEmail))
	if trimmed != "" {
		if actor, err := s.userRepo.FindUserByEmail(ctx, trimmed); err == nil && actor.IsSuperAdmin() {
			return actor.ID, nil
		}
	}

	approvedUsers, err := s.userRepo.ListUsersByStatus(ctx, domain.StatusApproved)
	if err == nil {
		for _, u := range approvedUsers {
			if u.IsSuperAdmin() {
				return u.ID, nil
			}
		}
	}

	return "", fmt.Errorf("%w: no approved SUPER_ADMIN found; run 'gestion-immo bootstrap super-admin' first", domain.ErrForbiddenSuperAdminOnly)
}

func (s *Service) requireSuperAdmin(ctx context.Context, actorUserID string) (*domain.User, error) {
	actor, err := s.userRepo.FindUserByID(ctx, strings.TrimSpace(actorUserID))
	if err != nil {
		return nil, domain.ErrForbiddenSuperAdminOnly
	}
	if !actor.IsSuperAdmin() {
		return nil, domain.ErrForbiddenSuperAdminOnly
	}
	return actor, nil
}

func (s *Service) issueInitialTokens(ctx context.Context, user domain.User) (AuthTokensDTO, error) {
	accessTok, err := s.jwtSvc.GenerateAccessToken(user.ID, user.Email, user.Role)
	if err != nil {
		return AuthTokensDTO{}, err
	}

	familyID, err := id.NewV7()
	if err != nil {
		return AuthTokensDTO{}, err
	}
	rawRefresh, record, err := s.buildRefreshTokenRecord(user.ID, familyID, time.Now().UTC())
	if err != nil {
		return AuthTokensDTO{}, err
	}
	if err := s.refreshRepo.SaveRefreshToken(ctx, record); err != nil {
		return AuthTokensDTO{}, err
	}

	return AuthTokensDTO{
		AccessToken:  accessTok,
		RefreshToken: rawRefresh,
		TokenType:    "Bearer",
		User:         toUserDTO(user),
	}, nil
}

func (s *Service) rotateTokens(ctx context.Context, user domain.User, oldRecord domain.RefreshTokenRecord, now time.Time) (AuthTokensDTO, error) {
	accessTok, err := s.jwtSvc.GenerateAccessToken(user.ID, user.Email, user.Role)
	if err != nil {
		return AuthTokensDTO{}, err
	}

	rawRefresh, newRecord, err := s.buildRefreshTokenRecord(user.ID, oldRecord.FamilyID, now)
	if err != nil {
		return AuthTokensDTO{}, err
	}
	if err := s.refreshRepo.RotateRefreshToken(ctx, oldRecord.ID, newRecord, now); err != nil {
		return AuthTokensDTO{}, err
	}

	return AuthTokensDTO{
		AccessToken:  accessTok,
		RefreshToken: rawRefresh,
		TokenType:    "Bearer",
		User:         toUserDTO(user),
	}, nil
}

// ValidateMCPBearer validates a raw Bearer token and returns the authenticated user ID,
// matching the mcpserver.TokenValidatorFunc signature.
func (s *Service) ValidateMCPBearer(ctx context.Context, rawBearer string) (string, error) {
	user, err := s.ValidateMCPToken(ctx, rawBearer)
	if err != nil {
		return "", err
	}
	return user.ID, nil
}

func (s *Service) buildRefreshTokenRecord(userID string, familyID string, now time.Time) (secret.String, domain.RefreshTokenRecord, error) {
	rawRefreshStr, err := secret.GenerateToken()
	if err != nil {
		return secret.String{}, domain.RefreshTokenRecord{}, err
	}
	rawRefresh := secret.New(rawRefreshStr)
	refreshHash := s.hasher.HashToken(rawRefresh)

	recID, err := id.NewV7()
	if err != nil {
		return secret.String{}, domain.RefreshTokenRecord{}, err
	}
	record := domain.RefreshTokenRecord{
		ID:        recID,
		UserID:    userID,
		FamilyID:  familyID,
		TokenHash: refreshHash,
		ExpiresAt: now.Add(defaultRefreshTTL),
		CreatedAt: now,
	}
	return rawRefresh, record, nil
}

func toUserDTO(u domain.User) UserDTO {
	return UserDTO{
		ID:         u.ID,
		FullName:   u.FullName,
		Email:      u.Email,
		Role:       u.Role,
		Status:     u.Status,
		ApprovedBy: u.ApprovedBy,
		ApprovedAt: u.ApprovedAt,
		CreatedAt:  u.CreatedAt,
	}
}

