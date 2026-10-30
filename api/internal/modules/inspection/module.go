package inspection

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yannamsellem/gestion-immo/internal/modules/inspection/application"
	"github.com/yannamsellem/gestion-immo/internal/modules/inspection/domain"
	"github.com/yannamsellem/gestion-immo/internal/modules/inspection/infrastructure/memory"
	insppostgres "github.com/yannamsellem/gestion-immo/internal/modules/inspection/infrastructure/postgres"
	insphttp "github.com/yannamsellem/gestion-immo/internal/modules/inspection/interfaces/http"
	inspmcp "github.com/yannamsellem/gestion-immo/internal/modules/inspection/interfaces/mcp"
	"github.com/yannamsellem/gestion-immo/internal/platform/mcpserver"
	"github.com/yannamsellem/gestion-immo/internal/platform/r2"
	"github.com/yannamsellem/gestion-immo/internal/platform/templating"
)

// Module encapsulates the inspection bounded context (repositories, UnitOfWork, application service, HTTP handler, and MCP tools).
type Module struct {
	svc     *application.Service
	handler *insphttp.Handler
}

// NewModule initializes the inspection Module from low-level platform dependencies (pool, storage, pdfEngine).
func NewModule(pool *pgxpool.Pool, storage r2.ObjectStorage, pdfEngine templating.PdfEngine) *Module {
	var (
		templates   domain.InspectionTemplateRepository
		inspections domain.InspectionRepository
		incidents   domain.IncidentRepository
		uow         domain.UnitOfWork
	)
	if pool != nil {
		templates = insppostgres.NewInspectionTemplateRepo(pool)
		inspections = insppostgres.NewInspectionRepo(pool, pool)
		incidents = insppostgres.NewIncidentRepo(pool)
		uow = insppostgres.NewUnitOfWork(pool)
	} else {
		templates = memory.NewInspectionTemplateRepo()
		inspections = memory.NewInspectionRepo()
		incidents = memory.NewIncidentRepo()
		uow = memory.NewUnitOfWork(templates, inspections, incidents)
	}

	svc := application.NewService(templates, inspections, incidents, uow, storage, pdfEngine)
	handler := insphttp.NewHandler(svc)
	return &Module{
		svc:     svc,
		handler: handler,
	}
}

// Service exposes the inspection application Service.
func (m *Module) Service() *application.Service {
	return m.svc
}

// RegisterHTTPRoutes mounts all inspection and incident REST HTTP routes onto mux.
func (m *Module) RegisterHTTPRoutes(mux *http.ServeMux) {
	m.handler.RegisterRoutes(mux)
}

// RegisterHumaRoutes mounts all inspection and incident REST routes onto api.
func (m *Module) RegisterHumaRoutes(api huma.API) {
	m.handler.RegisterHumaRoutes(api)
}

// RegisterMCPTools registers all inspection and incident MCP tools onto reg.
func (m *Module) RegisterMCPTools(reg *mcpserver.Registry) {
	inspmcp.RegisterTools(reg, m.svc)
}
