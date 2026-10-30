package lease

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yannamsellem/gestion-immo/internal/modules/lease/application"
	"github.com/yannamsellem/gestion-immo/internal/modules/lease/domain"
	"github.com/yannamsellem/gestion-immo/internal/modules/lease/infrastructure/memory"
	leasepostgres "github.com/yannamsellem/gestion-immo/internal/modules/lease/infrastructure/postgres"
	leasehttp "github.com/yannamsellem/gestion-immo/internal/modules/lease/interfaces/http"
	leasemcp "github.com/yannamsellem/gestion-immo/internal/modules/lease/interfaces/mcp"
	"github.com/yannamsellem/gestion-immo/internal/platform/mcpserver"
)

// Module encapsulates the lease bounded context (repositories, UnitOfWork, application service, HTTP handler, and MCP tools).
type Module struct {
	svc     *application.Service
	handler *leasehttp.Handler
}

// NewModule initializes the lease Module from low-level database dependencies.
func NewModule(pool *pgxpool.Pool) *Module {
	var (
		tenantRepo   domain.TenantRepository
		leaseRepo    domain.LeaseRepository
		rentCallRepo domain.RentCallRepository
		uow          domain.UnitOfWork
	)
	if pool != nil {
		tenantRepo = leasepostgres.NewTenantRepo(pool)
		leaseRepo = leasepostgres.NewLeaseRepo(pool)
		rentCallRepo = leasepostgres.NewRentCallRepo(pool, pool)
		uow = leasepostgres.NewUnitOfWork(pool)
	} else {
		tenantRepo = memory.NewTenantRepo()
		leaseRepo = memory.NewLeaseRepo()
		rentCallRepo = memory.NewRentCallRepo()
		uow = memory.NewUnitOfWork(tenantRepo, leaseRepo, rentCallRepo)
	}

	svc := application.NewService(tenantRepo, leaseRepo, rentCallRepo, uow)
	handler := leasehttp.NewHandler(svc)
	return &Module{
		svc:     svc,
		handler: handler,
	}
}

// Service exposes the lease application Service so dependent modules (payment, document) can consume it.
func (m *Module) Service() *application.Service {
	return m.svc
}

// RegisterHTTPRoutes mounts all lease REST HTTP routes onto mux.
func (m *Module) RegisterHTTPRoutes(mux *http.ServeMux) {
	m.handler.RegisterRoutes(mux)
}

// RegisterHumaRoutes mounts all lease REST routes onto api.
func (m *Module) RegisterHumaRoutes(api huma.API) {
	m.handler.RegisterHumaRoutes(api)
}

// RegisterMCPTools registers all lease MCP tools onto reg.
func (m *Module) RegisterMCPTools(reg *mcpserver.Registry) {
	leasemcp.RegisterTools(reg, m.svc)
}
