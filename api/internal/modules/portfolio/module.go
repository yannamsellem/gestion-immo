package portfolio

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yannamsellem/gestion-immo/internal/modules/portfolio/application"
	"github.com/yannamsellem/gestion-immo/internal/modules/portfolio/domain"
	"github.com/yannamsellem/gestion-immo/internal/modules/portfolio/infrastructure/memory"
	portfoliopostgres "github.com/yannamsellem/gestion-immo/internal/modules/portfolio/infrastructure/postgres"
	portfoliohttp "github.com/yannamsellem/gestion-immo/internal/modules/portfolio/interfaces/http"
	portfoliomcp "github.com/yannamsellem/gestion-immo/internal/modules/portfolio/interfaces/mcp"
	"github.com/yannamsellem/gestion-immo/internal/platform/mcpserver"
)

// Module encapsulates the portfolio bounded context (repositories, UnitOfWork, application service, HTTP handler, and MCP tools).
type Module struct {
	svc     *application.Service
	handler *portfoliohttp.Handler
}

// NewModule initializes the portfolio Module from low-level database dependencies.
func NewModule(pool *pgxpool.Pool) *Module {
	var (
		entityRepo      domain.LegalEntityRepository
		shareholderRepo domain.ShareholderRepository
		propertyRepo    domain.PropertyRepository
		lotRepo         domain.LotRepository
		uow             domain.UnitOfWork
	)
	if pool != nil {
		entityRepo = portfoliopostgres.NewLegalEntityRepo(pool)
		shareholderRepo = portfoliopostgres.NewShareholderRepo(pool)
		propertyRepo = portfoliopostgres.NewPropertyRepo(pool)
		lotRepo = portfoliopostgres.NewLotRepo(pool)
		uow = portfoliopostgres.NewUnitOfWork(pool)
	} else {
		entityRepo = memory.NewLegalEntityRepo()
		shareholderRepo = memory.NewShareholderRepo()
		propertyRepo = memory.NewPropertyRepo()
		lotRepo = memory.NewLotRepo()
		uow = memory.NewUnitOfWork(entityRepo, shareholderRepo, propertyRepo, lotRepo)
	}

	svc := application.NewService(entityRepo, shareholderRepo, propertyRepo, lotRepo, uow)
	handler := portfoliohttp.NewHandler(svc)
	return &Module{
		svc:     svc,
		handler: handler,
	}
}

// Service exposes the portfolio application Service so dependent modules (e.g. document) and CLI commands can consume it.
func (m *Module) Service() *application.Service {
	return m.svc
}

// RegisterHTTPRoutes mounts all portfolio REST HTTP routes onto mux.
func (m *Module) RegisterHTTPRoutes(mux *http.ServeMux) {
	m.handler.RegisterRoutes(mux)
}

// RegisterMCPTools registers all portfolio MCP tools onto reg.
func (m *Module) RegisterMCPTools(reg *mcpserver.Registry) {
	portfoliomcp.RegisterTools(reg, m.svc)
}
