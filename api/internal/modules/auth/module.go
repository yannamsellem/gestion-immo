package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yannamsellem/gestion-immo/internal/modules/auth/application"
	"github.com/yannamsellem/gestion-immo/internal/modules/auth/domain"
	"github.com/yannamsellem/gestion-immo/internal/modules/auth/infrastructure"
	"github.com/yannamsellem/gestion-immo/internal/modules/auth/infrastructure/memory"
	"github.com/yannamsellem/gestion-immo/internal/modules/auth/infrastructure/postgres"
	authhttp "github.com/yannamsellem/gestion-immo/internal/modules/auth/interfaces/http"
)

// Module encapsulates the auth bounded context (repositories, UnitOfWork, JWT & Argon2id services, application service, and HTTP handler).
type Module struct {
	authority *domain.TokenAuthority
	svc       *application.Service
	jwtSvc    *infrastructure.JwtService
	handler   *authhttp.Handler
}

// NewModule initializes the auth Module from low-level infrastructure dependencies.
func NewModule(pool *pgxpool.Pool, keysDir string, kid string) (*Module, error) {
	var (
		userRepo    domain.UserRepository
		refreshRepo domain.RefreshTokenRepository
		mcpRepo     domain.MCPTokenRepository
		uow         domain.UnitOfWork
	)
	if pool != nil {
		userRepo = postgres.NewUserRepo(pool)
		refreshRepo = postgres.NewRefreshTokenRepo(pool, pool)
		mcpRepo = postgres.NewMCPTokenRepo(pool)
		uow = postgres.NewUnitOfWork(pool)
	} else {
		userRepo = memory.NewUserRepo()
		refreshRepo = memory.NewRefreshTokenRepo()
		mcpRepo = memory.NewMCPTokenRepo()
		uow = memory.NewUnitOfWork(userRepo, refreshRepo, mcpRepo)
	}

	var (
		authority *domain.TokenAuthority
		err       error
	)
	if keysDir != "" {
		authority, err = domain.LoadTokenAuthorityFromDir(keysDir, kid)
	} else {
		authority, err = domain.GenerateTokenAuthority(kid)
	}
	if err != nil {
		return nil, fmt.Errorf("auth module: load token authority: %w", err)
	}

	jwtSvc, err := infrastructure.NewJwtService(authority, "gestion-immo", 15*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("auth module: init jwt service: %w", err)
	}
	hasher := infrastructure.NewArgon2Hasher()
	svc := application.NewService(userRepo, refreshRepo, mcpRepo, uow, jwtSvc, hasher)
	handler := authhttp.NewHandler(svc, jwtSvc)

	return &Module{
		authority: authority,
		svc:       svc,
		jwtSvc:    jwtSvc,
		handler:   handler,
	}, nil
}

// Service exposes the auth application Service for other modules, CLI commands, and MCP token validation.
func (m *Module) Service() *application.Service {
	return m.svc
}

// JwtService exposes the JwtService instance.
func (m *Module) JwtService() *infrastructure.JwtService {
	return m.jwtSvc
}

// Authority exposes the Ed25519 TokenAuthority.
func (m *Module) Authority() *domain.TokenAuthority {
	return m.authority
}

// RegisterHTTPRoutes registers all auth and SUPER_ADMIN user approval HTTP routes on mux.
func (m *Module) RegisterHTTPRoutes(mux *http.ServeMux) {
	m.handler.RegisterRoutes(mux)
}

// RegisterHumaRoutes registers all auth and SUPER_ADMIN user approval routes on api.
func (m *Module) RegisterHumaRoutes(api huma.API) {
	m.handler.RegisterHumaRoutes(api)
}

// RequireAuthMiddleware returns an HTTP middleware that validates Bearer JWTs and checks active user approval.
func (m *Module) RequireAuthMiddleware() func(http.Handler) http.Handler {
	return authhttp.RequireApprovedAuth(m.jwtSvc, func(ctx context.Context, userID string) error {
		_, err := m.svc.RequireApprovedUser(ctx, userID)
		return err
	})
}
