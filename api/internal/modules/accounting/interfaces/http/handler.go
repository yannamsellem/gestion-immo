package http

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/modules/accounting/application"
	"github.com/yannamsellem/gestion-immo/internal/modules/accounting/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/httpx"
)

// Handler exposes REST HTTP endpoints for the accounting bounded context
// (PCG journal entries, interactive financial statement previews with dynamic line toggling, Liasse Fiscale PDF, and DGFiP FEC export).
type Handler struct {
	svc *application.Service
}

// NewHandler constructs a new accounting HTTP Handler via constructor injection.
func NewHandler(svc *application.Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts all accounting REST endpoints on the Go 1.22+ ServeMux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/entities/{id}/journal-entries", h.handleRecordJournalEntry)
	mux.HandleFunc("GET /api/v1/entities/{id}/journal-entries", h.handleListJournalEntries)
	mux.HandleFunc("PUT /api/v1/entities/{id}/statements/lines", h.handleUpsertBalanceLines)
	mux.HandleFunc("POST /api/v1/entities/{id}/statements/preview", h.handlePreviewStatements)
	mux.HandleFunc("POST /api/v1/entities/{id}/statements/pdf", h.handleGenerateAnnualAccountsPDF)
	mux.HandleFunc("GET /api/v1/entities/{id}/fec", h.handleExportFEC)
}

type statementsRequest struct {
	FiscalYear      int      `json:"fiscal_year"`
	DisabledLineIDs []string `json:"disabled_line_ids"`
}

func (h *Handler) handleRecordJournalEntry(w http.ResponseWriter, r *http.Request) {
	var cmd application.RecordJournalEntryCmd
	if err := httpx.DecodeJSON(r, &cmd); err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}
	if strings.TrimSpace(cmd.LegalEntityID) == "" {
		cmd.LegalEntityID = r.PathValue("id")
	}

	dto, err := h.svc.RecordJournalEntry(r.Context(), cmd)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.Created(w, dto)
}

func (h *Handler) handleListJournalEntries(w http.ResponseWriter, r *http.Request) {
	entityID := r.PathValue("id")
	fiscalYear := parseFiscalYearQuery(r)
	list, err := h.svc.ListJournalEntries(r.Context(), entityID, fiscalYear)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if list == nil {
		list = []application.JournalEntryDTO{}
	}
	httpx.OK(w, list)
}

func (h *Handler) handleUpsertBalanceLines(w http.ResponseWriter, r *http.Request) {
	entityID := r.PathValue("id")
	var req struct {
		FiscalYear int                         `json:"fiscal_year"`
		Lines      []domain.AccountBalanceLine `json:"lines"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}
	if req.FiscalYear <= 0 {
		req.FiscalYear = parseFiscalYearQuery(r)
	}
	if err := h.svc.UpsertAccountBalanceLines(r.Context(), entityID, req.FiscalYear, req.Lines); err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handler) handlePreviewStatements(w http.ResponseWriter, r *http.Request) {
	entityID := r.PathValue("id")
	req, err := decodeOptionalStatementsRequest(r)
	if err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}
	if req.FiscalYear <= 0 {
		req.FiscalYear = parseFiscalYearQuery(r)
	}

	dto, err := h.svc.PreviewFinancialStatements(r.Context(), entityID, req.FiscalYear, req.DisabledLineIDs)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.OK(w, dto)
}

func (h *Handler) handleGenerateAnnualAccountsPDF(w http.ResponseWriter, r *http.Request) {
	entityID := r.PathValue("id")
	req, err := decodeOptionalStatementsRequest(r)
	if err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}
	if req.FiscalYear <= 0 {
		req.FiscalYear = parseFiscalYearQuery(r)
	}

	dto, err := h.svc.GenerateAnnualAccountsPDF(r.Context(), entityID, req.FiscalYear, req.DisabledLineIDs)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.Created(w, dto)
}

func (h *Handler) handleExportFEC(w http.ResponseWriter, r *http.Request) {
	entityID := r.PathValue("id")
	fiscalYear := parseFiscalYearQuery(r)

	fecBytes, err := h.svc.ExportEntityFEC(r.Context(), entityID, fiscalYear)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/tab-separated-values; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%sFEC%d1231.txt\"", strings.TrimSpace(entityID), fiscalYear))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(fecBytes)
}

func decodeOptionalStatementsRequest(r *http.Request) (statementsRequest, error) {
	var req statementsRequest
	if r.Body == nil {
		return req, nil
	}
	err := httpx.DecodeJSON(r, &req)
	if err != nil && !errors.Is(err, io.EOF) && !strings.Contains(err.Error(), "EOF") {
		return req, err
	}
	return req, nil
}

func parseFiscalYearQuery(r *http.Request) int {
	raw := strings.TrimSpace(r.URL.Query().Get("fiscal_year"))
	if raw == "" {
		return time.Now().UTC().Year()
	}
	val, err := strconv.Atoi(raw)
	if err != nil || val <= 0 {
		return time.Now().UTC().Year()
	}
	return val
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrValidation),
		errors.Is(err, domain.ErrUnbalancedJournalEntry):
		httpx.UnprocessableEntity(w, err.Error())
	case errors.Is(err, domain.ErrJournalEntryNotFound),
		errors.Is(err, domain.ErrReportNotFound):
		httpx.NotFound(w, err.Error())
	default:
		httpx.InternalError(w, "internal accounting error")
	}
}
