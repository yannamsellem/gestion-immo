package document

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yannamsellem/gestion-immo/internal/modules/document/application"
	"github.com/yannamsellem/gestion-immo/internal/modules/document/domain"
	"github.com/yannamsellem/gestion-immo/internal/modules/document/infrastructure/memory"
	docpostgres "github.com/yannamsellem/gestion-immo/internal/modules/document/infrastructure/postgres"
	dochttp "github.com/yannamsellem/gestion-immo/internal/modules/document/interfaces/http"
	docmcp "github.com/yannamsellem/gestion-immo/internal/modules/document/interfaces/mcp"
	"github.com/yannamsellem/gestion-immo/internal/platform/mcpserver"
	"github.com/yannamsellem/gestion-immo/internal/platform/r2"
	"github.com/yannamsellem/gestion-immo/internal/platform/templating"
)

// Module encapsulates the document bounded context (repositories, UnitOfWork, R2/Docx/PDF engines, application service, HTTP handler, and MCP tools).
type Module struct {
	svc     *application.Service
	handler *dochttp.Handler
}

// NewModule initializes the document Module from low-level platform components (pool, storage, pdfEngine)
// and the exposed domain ports from the portfolio, lease, and payment modules.
func NewModule(
	pool *pgxpool.Pool,
	storage r2.ObjectStorage,
	pdfEngine templating.PdfEngine,
	portfolioSvc application.PortfolioPort,
	leaseSvc application.LeasePort,
	paymentSvc application.PaymentPort,
) *Module {
	var (
		templates   domain.DocumentTemplateRepository
		storedFiles domain.StoredFileRepository
		gliDossiers domain.GLIDossierRepository
		uow         domain.UnitOfWork
	)
	if pool != nil {
		templates = docpostgres.NewDocumentTemplateRepo(pool)
		storedFiles = docpostgres.NewStoredFileRepo(pool)
		gliDossiers = docpostgres.NewGLIDossierRepo(pool)
		uow = docpostgres.NewUnitOfWork(pool)
	} else {
		templates = memory.NewDocumentTemplateRepo()
		storedFiles = memory.NewStoredFileRepo()
		gliDossiers = memory.NewGLIDossierRepo()
		uow = memory.NewUnitOfWork(templates, storedFiles, gliDossiers)
	}

	docxEngine := templating.NewDocxEngine()
	svc := application.NewService(
		templates,
		storedFiles,
		gliDossiers,
		uow,
		storage,
		docxEngine,
		pdfEngine,
		portfolioSvc,
		leaseSvc,
		paymentSvc,
	)
	handler := dochttp.NewHandler(svc)
	return &Module{
		svc:     svc,
		handler: handler,
	}
}

// Service exposes the document application Service.
func (m *Module) Service() *application.Service {
	return m.svc
}

// RegisterHTTPRoutes mounts all document REST HTTP routes onto mux.
func (m *Module) RegisterHTTPRoutes(mux *http.ServeMux) {
	m.handler.RegisterRoutes(mux)
}

// RegisterHumaRoutes mounts all document REST routes onto api.
func (m *Module) RegisterHumaRoutes(api huma.API) {
	m.handler.RegisterHumaRoutes(api)
}

// RegisterMCPTools registers all document MCP tools onto reg.
func (m *Module) RegisterMCPTools(reg *mcpserver.Registry) {
	docmcp.RegisterTools(reg, m.svc)
}
