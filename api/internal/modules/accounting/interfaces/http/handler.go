package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

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

type recordJournalEntryInput struct {
	ID   string `path:"id"`
	Body application.RecordJournalEntryCmd
}

type journalEntryOutput struct {
	Body application.JournalEntryDTO
}

type listJournalEntriesInput struct {
	ID         string `path:"id"`
	FiscalYear string `query:"fiscal_year" required:"false"`
}

type journalEntryListOutput struct {
	Body []application.JournalEntryDTO
}

// UpsertBalanceLinesBody represents the request body for upserting account balance lines.
type UpsertBalanceLinesBody struct {
	FiscalYear int                         `json:"fiscal_year,omitempty"`
	Lines      []domain.AccountBalanceLine `json:"lines"`
}

type upsertBalanceLinesInput struct {
	ID         string `path:"id"`
	FiscalYear string `query:"fiscal_year" required:"false"`
	Body       UpsertBalanceLinesBody
}

// StatementsRequestBody represents the optional request body for previewing or generating financial statements.
type StatementsRequestBody struct {
	FiscalYear      int      `json:"fiscal_year,omitempty"`
	DisabledLineIDs []string `json:"disabled_line_ids,omitempty"`
}

type statementsActionInput struct {
	ID         string `path:"id"`
	FiscalYear string `query:"fiscal_year" required:"false"`
	RawBody    []byte
}

type previewStatementsOutput struct {
	Body domain.ComputedFinancialStatementsDTO
}

type annualAccountsPDFOutput struct {
	Body application.AnnualAccountReportDTO
}

type exportFECInput struct {
	ID         string `path:"id"`
	FiscalYear string `query:"fiscal_year" required:"false"`
}

// RegisterRoutes mounts all accounting REST endpoints on the Go 1.22+ ServeMux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	h.RegisterHumaRoutes(httpx.NewHumaAPI(mux))
}

// RegisterHumaRoutes registers all accounting REST endpoints onto a Huma API.
func (h *Handler) RegisterHumaRoutes(api huma.API) {
	bearerSecurity := []map[string][]string{{"bearerAuth": {}}}

	huma.Register(api, huma.Operation{
		OperationID:   "recordJournalEntry",
		Method:        http.MethodPost,
		Path:          "/api/v1/entities/{id}/journal-entries",
		Summary:       "Record a balanced PCG double-entry journal entry",
		DefaultStatus: http.StatusCreated,
		Security:      bearerSecurity,
		Tags:          []string{"Accounting"},
	}, h.handleRecordJournalEntry)

	huma.Register(api, huma.Operation{
		OperationID:   "listJournalEntries",
		Method:        http.MethodGet,
		Path:          "/api/v1/entities/{id}/journal-entries",
		Summary:       "List PCG journal entries for a legal entity and fiscal year",
		DefaultStatus: http.StatusOK,
		Security:      bearerSecurity,
		Tags:          []string{"Accounting"},
	}, h.handleListJournalEntries)

	huma.Register(api, huma.Operation{
		OperationID:   "upsertBalanceLines",
		Method:        http.MethodPut,
		Path:          "/api/v1/entities/{id}/statements/lines",
		Summary:       "Upsert account balance lines for a legal entity and fiscal year",
		DefaultStatus: http.StatusNoContent,
		Security:      bearerSecurity,
		Tags:          []string{"Accounting"},
	}, h.handleUpsertBalanceLines)

	stmtBodySchema := huma.SchemaFromType(api.OpenAPI().Components.Schemas, reflect.TypeFor[StatementsRequestBody]())

	huma.Register(api, huma.Operation{
		OperationID:      "previewStatements",
		Method:           http.MethodPost,
		Path:             "/api/v1/entities/{id}/statements/preview",
		Summary:          "Preview dynamically computed financial statements (Balance Sheet, Income Statement, SIG)",
		DefaultStatus:    http.StatusOK,
		SkipValidateBody: true,
		Security:         bearerSecurity,
		Tags:             []string{"Accounting"},
	}, h.handlePreviewStatements)
	if pathItem := api.OpenAPI().Paths["/api/v1/entities/{id}/statements/preview"]; pathItem != nil && pathItem.Post != nil {
		pathItem.Post.RequestBody = &huma.RequestBody{
			Required: false,
			Content: map[string]*huma.MediaType{
				"application/json": {Schema: stmtBodySchema},
			},
		}
	}

	huma.Register(api, huma.Operation{
		OperationID:      "generateAnnualAccountsPdf",
		Method:           http.MethodPost,
		Path:             "/api/v1/entities/{id}/statements/pdf",
		Summary:          "Generate the Annual Accounts and Tax Return PDF package stored on R2",
		DefaultStatus:    http.StatusCreated,
		SkipValidateBody: true,
		Security:         bearerSecurity,
		Tags:             []string{"Accounting"},
	}, h.handleGenerateAnnualAccountsPDF)
	if pathItem := api.OpenAPI().Paths["/api/v1/entities/{id}/statements/pdf"]; pathItem != nil && pathItem.Post != nil {
		pathItem.Post.RequestBody = &huma.RequestBody{
			Required: false,
			Content: map[string]*huma.MediaType{
				"application/json": {Schema: stmtBodySchema},
			},
		}
	}

	huma.Register(api, huma.Operation{
		OperationID:   "exportEntityFec",
		Method:        http.MethodGet,
		Path:          "/api/v1/entities/{id}/fec",
		Summary:       "Export normative 18-column DGFiP FEC TSV file for a legal entity and fiscal year",
		DefaultStatus: http.StatusOK,
		Responses: map[string]*huma.Response{
			"200": {
				Description: "Normative 18-column DGFiP FEC TSV stream",
				Content: map[string]*huma.MediaType{
					"text/tab-separated-values": {
						Schema: &huma.Schema{Type: "string"},
					},
				},
			},
		},
		Security: bearerSecurity,
		Tags:     []string{"Accounting"},
	}, h.handleExportFEC)
}

func (h *Handler) handleRecordJournalEntry(ctx context.Context, in *recordJournalEntryInput) (*journalEntryOutput, error) {
	cmd := in.Body
	if strings.TrimSpace(cmd.LegalEntityID) == "" {
		cmd.LegalEntityID = in.ID
	}

	dto, err := h.svc.RecordJournalEntry(ctx, cmd)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &journalEntryOutput{Body: dto}, nil
}

func (h *Handler) handleListJournalEntries(ctx context.Context, in *listJournalEntriesInput) (*journalEntryListOutput, error) {
	fiscalYear := parseFiscalYearString(in.FiscalYear)
	list, err := h.svc.ListJournalEntries(ctx, in.ID, fiscalYear)
	if err != nil {
		return nil, mapDomainError(err)
	}
	if list == nil {
		list = []application.JournalEntryDTO{}
	}
	return &journalEntryListOutput{Body: list}, nil
}

func (h *Handler) handleUpsertBalanceLines(ctx context.Context, in *upsertBalanceLinesInput) (*struct{}, error) {
	fiscalYear := in.Body.FiscalYear
	if fiscalYear <= 0 {
		fiscalYear = parseFiscalYearString(in.FiscalYear)
	}
	if err := h.svc.UpsertAccountBalanceLines(ctx, in.ID, fiscalYear, in.Body.Lines); err != nil {
		return nil, mapDomainError(err)
	}
	return nil, nil
}

func (h *Handler) handlePreviewStatements(ctx context.Context, in *statementsActionInput) (*previewStatementsOutput, error) {
	req, err := decodeStatementsRequest(in.RawBody)
	if err != nil {
		return nil, httpx.StatusError(http.StatusBadRequest, err.Error())
	}
	if req.FiscalYear <= 0 {
		req.FiscalYear = parseFiscalYearString(in.FiscalYear)
	}

	dto, err := h.svc.PreviewFinancialStatements(ctx, in.ID, req.FiscalYear, req.DisabledLineIDs)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &previewStatementsOutput{Body: dto}, nil
}

func (h *Handler) handleGenerateAnnualAccountsPDF(ctx context.Context, in *statementsActionInput) (*annualAccountsPDFOutput, error) {
	req, err := decodeStatementsRequest(in.RawBody)
	if err != nil {
		return nil, httpx.StatusError(http.StatusBadRequest, err.Error())
	}
	if req.FiscalYear <= 0 {
		req.FiscalYear = parseFiscalYearString(in.FiscalYear)
	}

	dto, err := h.svc.GenerateAnnualAccountsPDF(ctx, in.ID, req.FiscalYear, req.DisabledLineIDs)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &annualAccountsPDFOutput{Body: dto}, nil
}

func (h *Handler) handleExportFEC(ctx context.Context, in *exportFECInput) (*huma.StreamResponse, error) {
	fiscalYear := parseFiscalYearString(in.FiscalYear)

	fecBytes, err := h.svc.ExportEntityFEC(ctx, in.ID, fiscalYear)
	if err != nil {
		return nil, mapDomainError(err)
	}

	safeEntityID := sanitizeFilenameToken(in.ID)
	return &huma.StreamResponse{
		Body: func(hctx huma.Context) {
			hctx.SetHeader("Content-Type", "text/tab-separated-values; charset=utf-8")
			hctx.SetHeader("X-Content-Type-Options", "nosniff")
			hctx.SetHeader("Content-Disposition", fmt.Sprintf("attachment; filename=\"%sFEC%d1231.txt\"", safeEntityID, fiscalYear))
			hctx.SetStatus(http.StatusOK)
			_, _ = hctx.BodyWriter().Write(fecBytes)
		},
	}, nil
}

func sanitizeFilenameToken(raw string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func decodeStatementsRequest(rawBody []byte) (StatementsRequestBody, error) {
	var req StatementsRequestBody
	trimmed := bytes.TrimSpace(rawBody)
	if len(trimmed) == 0 {
		return req, nil
	}
	if err := json.Unmarshal(trimmed, &req); err != nil {
		return req, err
	}
	return req, nil
}

func parseFiscalYearString(raw string) int {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Now().UTC().Year()
	}
	val, err := strconv.Atoi(trimmed)
	if err != nil || val <= 0 {
		return time.Now().UTC().Year()
	}
	return val
}

func mapDomainError(err error) error {
	switch {
	case errors.Is(err, domain.ErrValidation),
		errors.Is(err, domain.ErrUnbalancedJournalEntry):
		return httpx.StatusError(http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, domain.ErrJournalEntryNotFound),
		errors.Is(err, domain.ErrReportNotFound):
		return httpx.StatusError(http.StatusNotFound, err.Error())
	default:
		return httpx.StatusError(http.StatusInternalServerError, "internal accounting error")
	}
}
