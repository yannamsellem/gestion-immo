package accounting

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yannamsellem/gestion-immo/internal/modules/accounting/application"
	"github.com/yannamsellem/gestion-immo/internal/modules/accounting/domain"
	"github.com/yannamsellem/gestion-immo/internal/modules/accounting/infrastructure"
	"github.com/yannamsellem/gestion-immo/internal/modules/accounting/infrastructure/memory"
	accpostgres "github.com/yannamsellem/gestion-immo/internal/modules/accounting/infrastructure/postgres"
	acchttp "github.com/yannamsellem/gestion-immo/internal/modules/accounting/interfaces/http"
	accmcp "github.com/yannamsellem/gestion-immo/internal/modules/accounting/interfaces/mcp"
	"github.com/yannamsellem/gestion-immo/internal/platform/mcpserver"
	"github.com/yannamsellem/gestion-immo/internal/platform/r2"
	"github.com/yannamsellem/gestion-immo/internal/platform/templating"
)

// Module encapsulates the accounting bounded context (repositories, UnitOfWork, FEC/Liasse adapters, application service, HTTP handler, and MCP tools).
type Module struct {
	svc     *application.Service
	handler *acchttp.Handler
}

// NewModule initializes the accounting Module from low-level platform dependencies (pool, storage, pdfEngine) and an optional PortfolioPort.
func NewModule(pool *pgxpool.Pool, storage r2.ObjectStorage, pdfEngine templating.PdfEngine, portfolioPorts ...application.PortfolioPort) *Module {
	var (
		journalEntries  domain.JournalEntryRepository
		accountBalances domain.AccountBalanceRepository
		annualReports   domain.AnnualAccountReportRepository
		uow             domain.UnitOfWork
	)
	if pool != nil {
		journalEntries = accpostgres.NewJournalEntryRepo(pool, pool)
		accountBalances = accpostgres.NewAccountBalanceRepo(pool, pool)
		annualReports = accpostgres.NewAnnualAccountReportRepo(pool)
		uow = accpostgres.NewUnitOfWork(pool)
	} else {
		journalEntries = memory.NewJournalEntryRepo()
		accountBalances = memory.NewAccountBalanceRepo()
		annualReports = memory.NewAnnualAccountReportRepo()
		uow = memory.NewUnitOfWork(journalEntries, accountBalances, annualReports)
	}

	fecExporter := infrastructure.NewFECExporter()
	annualAccountsRenderer := infrastructure.NewAnnualAccountsPDFRenderer(pdfEngine)

	svc := application.NewService(
		journalEntries,
		accountBalances,
		annualReports,
		uow,
		storage,
		pdfEngine,
		fecExporter,
		annualAccountsRenderer,
		portfolioPorts...,
	)
	handler := acchttp.NewHandler(svc)
	return &Module{
		svc:     svc,
		handler: handler,
	}
}

// Service exposes the accounting application Service.
func (m *Module) Service() *application.Service {
	return m.svc
}

// RegisterHTTPRoutes mounts all accounting REST HTTP routes onto mux.
func (m *Module) RegisterHTTPRoutes(mux *http.ServeMux) {
	m.handler.RegisterRoutes(mux)
}

// RegisterHumaRoutes mounts all accounting REST routes onto api.
func (m *Module) RegisterHumaRoutes(api huma.API) {
	m.handler.RegisterHumaRoutes(api)
}

// RegisterMCPTools registers all accounting MCP tools onto reg.
func (m *Module) RegisterMCPTools(reg *mcpserver.Registry) {
	accmcp.RegisterTools(reg, m.svc)
}
