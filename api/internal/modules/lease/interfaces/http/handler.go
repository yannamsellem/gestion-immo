package http

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/modules/lease/application"
	"github.com/yannamsellem/gestion-immo/internal/modules/lease/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/httpx"
)

// Handler exposes REST HTTP endpoints for the lease bounded context.
type Handler struct {
	svc *application.Service
}

// NewHandler constructs a new lease HTTP Handler via constructor injection.
func NewHandler(svc *application.Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts all lease REST endpoints on the Go 1.22+ ServeMux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/tenants", h.handleCreateTenant)
	mux.HandleFunc("GET /api/v1/tenants", h.handleListTenants)
	mux.HandleFunc("POST /api/v1/tenants/{id}/archive", h.handleArchiveTenant)
	mux.HandleFunc("POST /api/v1/leases", h.handleCreateLease)
	mux.HandleFunc("GET /api/v1/leases", h.handleListLeases)
	mux.HandleFunc("POST /api/v1/entities/{id}/rent-calls/generate", h.handleGenerateRentCalls)
	mux.HandleFunc("GET /api/v1/entities/{id}/rent-calls", h.handleListEntityRentCalls)
}

func (h *Handler) handleCreateTenant(w http.ResponseWriter, r *http.Request) {
	var cmd application.CreateTenantCmd
	if err := httpx.DecodeJSON(r, &cmd); err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}

	dto, err := h.svc.CreateTenant(r.Context(), cmd)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.Created(w, dto)
}

func (h *Handler) handleListTenants(w http.ResponseWriter, r *http.Request) {
	includeArchived := strings.EqualFold(r.URL.Query().Get("include_archived"), "true") ||
		r.URL.Query().Get("include_archived") == "1"

	tenants, err := h.svc.ListTenants(r.Context(), includeArchived)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if tenants == nil {
		tenants = []application.TenantDTO{}
	}

	httpx.OK(w, tenants)
}

func (h *Handler) handleArchiveTenant(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("id")
	if err := h.svc.ArchiveTenant(r.Context(), tenantID); err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.NoContent(w)
}

func (h *Handler) handleCreateLease(w http.ResponseWriter, r *http.Request) {
	var cmd application.CreateLeaseCmd
	if err := httpx.DecodeJSON(r, &cmd); err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}

	dto, err := h.svc.CreateLease(r.Context(), cmd)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.Created(w, dto)
}

func (h *Handler) handleListLeases(w http.ResponseWriter, r *http.Request) {
	entityID := strings.TrimSpace(r.URL.Query().Get("legal_entity_id"))
	if entityID == "" {
		entityID = strings.TrimSpace(r.URL.Query().Get("entity_id"))
	}

	var (
		leases []application.LeaseDTO
		err    error
	)
	if entityID != "" {
		leases, err = h.svc.ListLeasesByEntity(r.Context(), entityID)
	} else {
		leases, err = h.svc.ListLeases(r.Context())
	}
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if leases == nil {
		leases = []application.LeaseDTO{}
	}

	httpx.OK(w, leases)
}

func (h *Handler) handleGenerateRentCalls(w http.ResponseWriter, r *http.Request) {
	entityID := r.PathValue("id")
	periodMonth := strings.TrimSpace(r.URL.Query().Get("period_month"))

	var body struct {
		PeriodMonth string `json:"period_month"`
	}
	if r.Body != nil {
		err := httpx.DecodeJSON(r, &body)
		if err != nil && !errors.Is(err, io.EOF) {
			httpx.BadRequest(w, err.Error())
			return
		}
		if strings.TrimSpace(body.PeriodMonth) != "" {
			periodMonth = strings.TrimSpace(body.PeriodMonth)
		}
	}
	if periodMonth == "" {
		periodMonth = time.Now().UTC().Format("2006-01")
	}

	calls, err := h.svc.GenerateMonthlyRentCalls(r.Context(), entityID, periodMonth)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if calls == nil {
		calls = []application.RentCallDTO{}
	}

	httpx.Created(w, calls)
}

func (h *Handler) handleListEntityRentCalls(w http.ResponseWriter, r *http.Request) {
	entityID := r.PathValue("id")
	calls, err := h.svc.ListRentCallsByEntity(r.Context(), entityID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if calls == nil {
		calls = []application.RentCallDTO{}
	}

	httpx.OK(w, calls)
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrValidation):
		httpx.UnprocessableEntity(w, err.Error())
	case errors.Is(err, domain.ErrDuplicatePeriod):
		httpx.Conflict(w, err.Error())
	case errors.Is(err, domain.ErrTenantNotFound),
		errors.Is(err, domain.ErrLeaseNotFound),
		errors.Is(err, domain.ErrRentCallNotFound):
		httpx.NotFound(w, err.Error())
	default:
		httpx.InternalError(w, "internal lease error")
	}
}
