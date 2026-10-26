package payment

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yannamsellem/gestion-immo/internal/modules/payment/application"
	"github.com/yannamsellem/gestion-immo/internal/modules/payment/domain"
	"github.com/yannamsellem/gestion-immo/internal/modules/payment/infrastructure"
	"github.com/yannamsellem/gestion-immo/internal/modules/payment/infrastructure/memory"
	paymentpostgres "github.com/yannamsellem/gestion-immo/internal/modules/payment/infrastructure/postgres"
	paymenthttp "github.com/yannamsellem/gestion-immo/internal/modules/payment/interfaces/http"
	paymentmcp "github.com/yannamsellem/gestion-immo/internal/modules/payment/interfaces/mcp"
	"github.com/yannamsellem/gestion-immo/internal/platform/mcpserver"
)

// Module encapsulates the payment bounded context (repositories, UnitOfWork, CSV/DSP2 adapters, application service, HTTP handler, and MCP tools).
type Module struct {
	svc     *application.Service
	handler *paymenthttp.Handler
}

// NewModule initializes the payment Module from low-level database dependencies and the exposed LeasePort from the lease module.
func NewModule(pool *pgxpool.Pool, leaseSvc application.LeasePort, enableOpenBanking bool) *Module {
	var (
		payments    domain.PaymentRepository
		cashJournal domain.CashJournalRepository
		uow         domain.UnitOfWork
	)
	if pool != nil {
		payments = paymentpostgres.NewPaymentRepo(pool, pool)
		cashJournal = paymentpostgres.NewCashJournalRepo(pool)
		uow = paymentpostgres.NewUnitOfWork(pool)
	} else {
		payments = memory.NewPaymentRepo()
		cashJournal = memory.NewCashJournalRepo()
		uow = memory.NewUnitOfWork(payments, cashJournal)
	}

	bankProvider := infrastructure.NewDSP2Provider(enableOpenBanking, "dsp2")
	csvImporter := infrastructure.NewCSVImporter()

	svc := application.NewService(payments, cashJournal, uow, leaseSvc, bankProvider, csvImporter)
	handler := paymenthttp.NewHandler(svc)
	return &Module{
		svc:     svc,
		handler: handler,
	}
}

// Service exposes the payment application Service so dependent modules (e.g. document) and CLI commands can consume it.
func (m *Module) Service() *application.Service {
	return m.svc
}

// RegisterHTTPRoutes mounts all payment REST HTTP routes onto mux.
func (m *Module) RegisterHTTPRoutes(mux *http.ServeMux) {
	m.handler.RegisterRoutes(mux)
}

// RegisterMCPTools registers all payment MCP tools onto reg.
func (m *Module) RegisterMCPTools(reg *mcpserver.Registry) {
	paymentmcp.RegisterTools(reg, m.svc)
}
