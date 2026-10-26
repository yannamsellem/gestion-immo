package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

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

// RegisterRoutes mounts all payment REST endpoints on the Go 1.22+ ServeMux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/payments/cash", h.handleRecordCashPayment)
	mux.HandleFunc("POST /api/v1/payments/bank", h.handleRecordBankPayment)
	mux.HandleFunc("POST /api/v1/payments/cash-deposit", h.handleRecordCashDeposit)
	mux.HandleFunc("POST /api/v1/payments/import-csv", h.handleImportPaymentsCSV)
	mux.HandleFunc("GET /api/v1/entities/{id}/negative-positions", h.handleListNegativePositions)
	mux.HandleFunc("GET /api/v1/entities/{id}/cash-journal", h.handleListCashJournal)
}

func (h *Handler) handleRecordCashPayment(w http.ResponseWriter, r *http.Request) {
	var cmd application.RecordCashPaymentCmd
	if err := httpx.DecodeJSON(r, &cmd); err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}

	dto, err := h.svc.RecordCashPayment(r.Context(), cmd)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.Created(w, dto)
}

func (h *Handler) handleRecordBankPayment(w http.ResponseWriter, r *http.Request) {
	var cmd application.RecordBankPaymentCmd
	if err := httpx.DecodeJSON(r, &cmd); err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}

	dto, err := h.svc.RecordBankPayment(r.Context(), cmd)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.Created(w, dto)
}

func (h *Handler) handleRecordCashDeposit(w http.ResponseWriter, r *http.Request) {
	var cmd application.RecordCashDepositCmd
	if err := httpx.DecodeJSON(r, &cmd); err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}

	dto, err := h.svc.RecordCashDepositToBank(r.Context(), cmd)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.Created(w, dto)
}

func (h *Handler) handleImportPaymentsCSV(w http.ResponseWriter, r *http.Request) {
	entityID := strings.TrimSpace(r.URL.Query().Get("entity_id"))
	if entityID == "" {
		entityID = strings.TrimSpace(r.URL.Query().Get("legal_entity_id"))
	}

	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	var csvReader io.Reader

	switch {
	case strings.Contains(contentType, "multipart/form-data"):
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			httpx.BadRequest(w, "invalid multipart form")
			return
		}
		if entityID == "" {
			entityID = strings.TrimSpace(r.FormValue("entity_id"))
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			httpx.BadRequest(w, "missing CSV file field")
			return
		}
		defer file.Close()
		raw, err := io.ReadAll(file)
		if err != nil {
			httpx.BadRequest(w, "failed to read uploaded file")
			return
		}
		csvReader = bytes.NewReader(raw)

	case strings.Contains(contentType, "application/json"):
		var body struct {
			EntityID      string `json:"entity_id"`
			LegalEntityID string `json:"legal_entity_id"`
			CSVContent    string `json:"csv_content"`
			CSV           string `json:"csv"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpx.BadRequest(w, "invalid JSON payload")
			return
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
		if r.Body == nil {
			httpx.BadRequest(w, "empty request body")
			return
		}
		defer r.Body.Close()
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			httpx.BadRequest(w, "failed to read request body")
			return
		}
		csvReader = bytes.NewReader(raw)
	}

	res, err := h.svc.ImportPaymentsCSV(r.Context(), entityID, csvReader)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.OK(w, res)
}

func (h *Handler) handleListNegativePositions(w http.ResponseWriter, r *http.Request) {
	entityID := r.PathValue("id")
	positions, err := h.svc.ListNegativePositions(r.Context(), entityID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if positions == nil {
		positions = []application.NegativePositionDTO{}
	}

	httpx.OK(w, positions)
}

func (h *Handler) handleListCashJournal(w http.ResponseWriter, r *http.Request) {
	entityID := r.PathValue("id")
	entries, err := h.svc.ListCashJournalByEntity(r.Context(), entityID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if entries == nil {
		entries = []application.CashJournalEntryDTO{}
	}

	httpx.OK(w, entries)
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrValidation), errors.Is(err, leasedomain.ErrValidation):
		httpx.UnprocessableEntity(w, err.Error())
	case errors.Is(err, domain.ErrOpenBankingDisabled):
		httpx.Conflict(w, err.Error())
	case errors.Is(err, domain.ErrPaymentNotFound),
		errors.Is(err, leasedomain.ErrLeaseNotFound),
		errors.Is(err, leasedomain.ErrTenantNotFound),
		errors.Is(err, leasedomain.ErrRentCallNotFound):
		httpx.NotFound(w, err.Error())
	default:
		httpx.InternalError(w, "internal payment error")
	}
}
