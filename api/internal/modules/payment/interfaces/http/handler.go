package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"reflect"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	leasedomain "github.com/yannamsellem/gestion-immo/internal/modules/lease/domain"
	"github.com/yannamsellem/gestion-immo/internal/modules/payment/application"
	"github.com/yannamsellem/gestion-immo/internal/modules/payment/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/httpx"
)

// Handler exposes REST HTTP endpoints for the payment bounded context.
type Handler struct {
	svc *application.Service
}

// NewHandler constructs a new payment HTTP Handler via constructor injection.
func NewHandler(svc *application.Service) *Handler {
	return &Handler{svc: svc}
}

type recordCashPaymentInput struct {
	Body application.RecordCashPaymentCmd
}

type recordBankPaymentInput struct {
	Body application.RecordBankPaymentCmd
}

type paymentReceiptOutput struct {
	Body application.PaymentReceiptDTO
}

type recordCashDepositInput struct {
	Body application.RecordCashDepositCmd
}

type cashJournalEntryOutput struct {
	Body application.CashJournalEntryDTO
}

// ImportPaymentsCSVBody represents the JSON request body for importing CSV/OFX payments.
type ImportPaymentsCSVBody struct {
	EntityID      string `json:"entity_id" format:"uuid"`
	LegalEntityID string `json:"legal_entity_id,omitempty" format:"uuid"`
	CSVContent    string `json:"csv_content"`
	CSV           string `json:"csv,omitempty"`
}

type importPaymentsCSVInput struct {
	EntityID      string `query:"entity_id" required:"false"`
	LegalEntityID string `query:"legal_entity_id" required:"false"`
	ContentType   string `header:"Content-Type"`
	RawBody       []byte
}

type batchImportResultOutput struct {
	Body application.BatchImportResultDTO
}

type entityIDInput struct {
	ID string `path:"id" format:"uuid"`
}

type negativePositionListOutput struct {
	Body []application.NegativePositionDTO
}

type cashJournalEntryListOutput struct {
	Body []application.CashJournalEntryDTO
}

// RegisterRoutes mounts all payment REST endpoints on the Go 1.22+ ServeMux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	h.RegisterHumaRoutes(httpx.NewHumaAPI(mux))
}

// RegisterHumaRoutes registers all payment REST endpoints onto a Huma API.
func (h *Handler) RegisterHumaRoutes(api huma.API) {
	bearerSecurity := []map[string][]string{{"bearerAuth": {}}}

	huma.Register(api, huma.Operation{
		OperationID:   "recordCashPayment",
		Method:        http.MethodPost,
		Path:          "/api/v1/payments/cash",
		Summary:       "Record a cash rent payment (PCG 530 Caisse)",
		DefaultStatus: http.StatusCreated,
		Security:      bearerSecurity,
		Tags:          []string{"Payment"},
	}, h.handleRecordCashPayment)

	huma.Register(api, huma.Operation{
		OperationID:   "recordBankPayment",
		Method:        http.MethodPost,
		Path:          "/api/v1/payments/bank",
		Summary:       "Record a bank transfer, check, or direct debit rent payment",
		DefaultStatus: http.StatusCreated,
		Security:      bearerSecurity,
		Tags:          []string{"Payment"},
	}, h.handleRecordBankPayment)

	huma.Register(api, huma.Operation{
		OperationID:   "recordCashDeposit",
		Method:        http.MethodPost,
		Path:          "/api/v1/payments/cash-deposit",
		Summary:       "Record a cash deposit from PCG 530 (Caisse) to PCG 580/512 (Banque)",
		DefaultStatus: http.StatusCreated,
		Security:      bearerSecurity,
		Tags:          []string{"Payment"},
	}, h.handleRecordCashDeposit)

	csvBodySchema := huma.SchemaFromType(api.OpenAPI().Components.Schemas, reflect.TypeFor[ImportPaymentsCSVBody]())
	huma.Register(api, huma.Operation{
		OperationID:      "importPaymentsCsv",
		Method:           http.MethodPost,
		Path:             "/api/v1/payments/import-csv",
		Summary:          "Import a batch of bank payments from CSV or OFX content",
		DefaultStatus:    http.StatusOK,
		MaxBodyBytes:     httpx.MaxUploadBodyBytes,
		SkipValidateBody: true,
		Security:         bearerSecurity,
		Tags:             []string{"Payment"},
	}, h.handleImportPaymentsCSV)
	if pathItem := api.OpenAPI().Paths["/api/v1/payments/import-csv"]; pathItem != nil && pathItem.Post != nil {
		pathItem.Post.RequestBody = &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {Schema: csvBodySchema},
				"text/csv":         {Schema: &huma.Schema{Type: "string"}},
			},
		}
	}

	huma.Register(api, huma.Operation{
		OperationID:   "listNegativePositions",
		Method:        http.MethodGet,
		Path:          "/api/v1/entities/{id}/negative-positions",
		Summary:       "List negative tenant positions (unpaid rent calls) for a legal entity",
		DefaultStatus: http.StatusOK,
		Security:      bearerSecurity,
		Tags:          []string{"Payment"},
	}, h.handleListNegativePositions)

	huma.Register(api, huma.Operation{
		OperationID:   "listCashJournal",
		Method:        http.MethodGet,
		Path:          "/api/v1/entities/{id}/cash-journal",
		Summary:       "List PCG 530 Cash Journal entries for a legal entity",
		DefaultStatus: http.StatusOK,
		Security:      bearerSecurity,
		Tags:          []string{"Payment"},
	}, h.handleListCashJournal)
}

func (h *Handler) handleRecordCashPayment(ctx context.Context, in *recordCashPaymentInput) (*paymentReceiptOutput, error) {
	dto, err := h.svc.RecordCashPayment(ctx, in.Body)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &paymentReceiptOutput{Body: dto}, nil
}

func (h *Handler) handleRecordBankPayment(ctx context.Context, in *recordBankPaymentInput) (*paymentReceiptOutput, error) {
	dto, err := h.svc.RecordBankPayment(ctx, in.Body)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &paymentReceiptOutput{Body: dto}, nil
}

func (h *Handler) handleRecordCashDeposit(ctx context.Context, in *recordCashDepositInput) (*cashJournalEntryOutput, error) {
	dto, err := h.svc.RecordCashDepositToBank(ctx, in.Body)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &cashJournalEntryOutput{Body: dto}, nil
}

func (h *Handler) handleImportPaymentsCSV(ctx context.Context, in *importPaymentsCSVInput) (*batchImportResultOutput, error) {
	entityID := strings.TrimSpace(in.EntityID)
	if entityID == "" {
		entityID = strings.TrimSpace(in.LegalEntityID)
	}

	contentType := strings.ToLower(in.ContentType)
	var csvReader io.Reader

	switch {
	case strings.Contains(contentType, "multipart/form-data"):
		_, params, err := mime.ParseMediaType(in.ContentType)
		if err != nil || params["boundary"] == "" {
			return nil, httpx.StatusError(http.StatusBadRequest, "invalid multipart form")
		}
		mr := multipart.NewReader(bytes.NewReader(in.RawBody), params["boundary"])
		form, err := mr.ReadForm(10 << 20)
		if err != nil {
			return nil, httpx.StatusError(http.StatusBadRequest, "invalid multipart form")
		}
		defer func() { _ = form.RemoveAll() }()
		if entityID == "" && len(form.Value["entity_id"]) > 0 {
			entityID = strings.TrimSpace(form.Value["entity_id"][0])
		}
		files := form.File["file"]
		if len(files) == 0 {
			return nil, httpx.StatusError(http.StatusBadRequest, "missing CSV file field")
		}
		file, err := files[0].Open()
		if err != nil {
			return nil, httpx.StatusError(http.StatusBadRequest, "failed to read uploaded file")
		}
		defer file.Close()
		raw, err := io.ReadAll(io.LimitReader(file, httpx.MaxUploadBodyBytes))
		if err != nil {
			return nil, httpx.StatusError(http.StatusBadRequest, "failed to read uploaded file")
		}
		csvReader = bytes.NewReader(raw)

	case strings.Contains(contentType, "application/json") || (contentType == "" && bytes.HasPrefix(bytes.TrimSpace(in.RawBody), []byte("{"))):
		var body ImportPaymentsCSVBody
		if err := json.Unmarshal(in.RawBody, &body); err != nil {
			return nil, httpx.StatusError(http.StatusBadRequest, "invalid JSON payload")
		}
		if entityID == "" {
			entityID = strings.TrimSpace(body.EntityID)
			if entityID == "" {
				entityID = strings.TrimSpace(body.LegalEntityID)
			}
		}
		content := body.CSVContent
		if content == "" {
			content = body.CSV
		}
		csvReader = strings.NewReader(content)

	default:
		if len(in.RawBody) == 0 {
			return nil, httpx.StatusError(http.StatusBadRequest, "empty request body")
		}
		csvReader = bytes.NewReader(in.RawBody)
	}

	res, err := h.svc.ImportPaymentsCSV(ctx, entityID, csvReader)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &batchImportResultOutput{Body: res}, nil
}

func (h *Handler) handleListNegativePositions(ctx context.Context, in *entityIDInput) (*negativePositionListOutput, error) {
	positions, err := h.svc.ListNegativePositions(ctx, in.ID)
	if err != nil {
		return nil, mapDomainError(err)
	}
	if positions == nil {
		positions = []application.NegativePositionDTO{}
	}
	return &negativePositionListOutput{Body: positions}, nil
}

func (h *Handler) handleListCashJournal(ctx context.Context, in *entityIDInput) (*cashJournalEntryListOutput, error) {
	entries, err := h.svc.ListCashJournalByEntity(ctx, in.ID)
	if err != nil {
		return nil, mapDomainError(err)
	}
	if entries == nil {
		entries = []application.CashJournalEntryDTO{}
	}
	return &cashJournalEntryListOutput{Body: entries}, nil
}

func mapDomainError(err error) error {
	switch {
	case errors.Is(err, domain.ErrValidation), errors.Is(err, leasedomain.ErrValidation):
		return httpx.StatusError(http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, domain.ErrOpenBankingDisabled):
		return httpx.StatusError(http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrPaymentNotFound),
		errors.Is(err, leasedomain.ErrLeaseNotFound),
		errors.Is(err, leasedomain.ErrTenantNotFound),
		errors.Is(err, leasedomain.ErrRentCallNotFound):
		return httpx.StatusError(http.StatusNotFound, err.Error())
	default:
		return httpx.StatusError(http.StatusInternalServerError, "internal payment error")
	}
}
