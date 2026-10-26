package http

import (
	"errors"
	"net/http"

	"github.com/yannamsellem/gestion-immo/internal/modules/portfolio/application"
	"github.com/yannamsellem/gestion-immo/internal/modules/portfolio/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/httpx"
)

// Handler exposes REST HTTP endpoints for the portfolio bounded context.
type Handler struct {
	svc *application.Service
}

// NewHandler constructs a new portfolio HTTP Handler via constructor injection.
func NewHandler(svc *application.Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts all portfolio REST endpoints on the Go 1.22+ ServeMux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/entities", h.handleCreateEntity)
	mux.HandleFunc("GET /api/v1/entities", h.handleListEntities)
	mux.HandleFunc("GET /api/v1/entities/{id}", h.handleGetEntitySummary)
	mux.HandleFunc("POST /api/v1/entities/{id}/shareholders", h.handleUpsertShareholder)
	mux.HandleFunc("POST /api/v1/entities/{id}/properties", h.handleCreateProperty)
	mux.HandleFunc("GET /api/v1/entities/{id}/properties", h.handleListProperties)
	mux.HandleFunc("POST /api/v1/properties/{id}/lots", h.handleCreateLot)
}

func (h *Handler) handleCreateEntity(w http.ResponseWriter, r *http.Request) {
	var cmd application.CreateLegalEntityCmd
	if err := httpx.DecodeJSON(r, &cmd); err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}

	dto, err := h.svc.CreateLegalEntity(r.Context(), cmd)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.Created(w, dto)
}

func (h *Handler) handleListEntities(w http.ResponseWriter, r *http.Request) {
	entities, err := h.svc.ListLegalEntities(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if entities == nil {
		entities = []application.LegalEntityDTO{}
	}

	httpx.OK(w, entities)
}

func (h *Handler) handleGetEntitySummary(w http.ResponseWriter, r *http.Request) {
	entityID := r.PathValue("id")
	summary, err := h.svc.GetEntityPortfolioSummary(r.Context(), entityID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.OK(w, summary)
}

func (h *Handler) handleUpsertShareholder(w http.ResponseWriter, r *http.Request) {
	var cmd application.UpsertShareholderCmd
	if err := httpx.DecodeJSON(r, &cmd); err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}
	cmd.LegalEntityID = r.PathValue("id")

	dto, err := h.svc.UpsertShareholder(r.Context(), cmd)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.OK(w, dto)
}

func (h *Handler) handleCreateProperty(w http.ResponseWriter, r *http.Request) {
	var cmd application.CreatePropertyCmd
	if err := httpx.DecodeJSON(r, &cmd); err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}
	cmd.LegalEntityID = r.PathValue("id")

	dto, err := h.svc.CreateProperty(r.Context(), cmd)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.Created(w, dto)
}

func (h *Handler) handleListProperties(w http.ResponseWriter, r *http.Request) {
	entityID := r.PathValue("id")
	props, err := h.svc.ListProperties(r.Context(), entityID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if props == nil {
		props = []application.PropertyDTO{}
	}

	httpx.OK(w, props)
}

func (h *Handler) handleCreateLot(w http.ResponseWriter, r *http.Request) {
	var cmd application.CreateLotCmd
	if err := httpx.DecodeJSON(r, &cmd); err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}
	cmd.PropertyID = r.PathValue("id")

	dto, err := h.svc.CreateLot(r.Context(), cmd)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.Created(w, dto)
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrValidation):
		httpx.UnprocessableEntity(w, err.Error())
	case errors.Is(err, domain.ErrDuplicateSIRET):
		httpx.Conflict(w, err.Error())
	case errors.Is(err, domain.ErrLegalEntityNotFound),
		errors.Is(err, domain.ErrPropertyNotFound),
		errors.Is(err, domain.ErrLotNotFound),
		errors.Is(err, domain.ErrShareholderNotFound):
		httpx.NotFound(w, err.Error())
	default:
		httpx.InternalError(w, "internal portfolio error")
	}
}
