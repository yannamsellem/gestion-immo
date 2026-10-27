package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/yannamsellem/gestion-immo/internal/modules/inspection/application"
	"github.com/yannamsellem/gestion-immo/internal/modules/inspection/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/httpx"
)

// Handler exposes REST HTTP endpoints for the inspection bounded context (templates, move-in/move-out inspections, and incidents).
type Handler struct {
	svc *application.Service
}

// NewHandler constructs a new inspection HTTP Handler via constructor injection.
func NewHandler(svc *application.Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts all inspection and incident REST endpoints on the Go 1.22+ ServeMux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/inspection-templates", h.handleCreateTemplate)
	mux.HandleFunc("GET /api/v1/inspection-templates", h.handleListTemplates)
	mux.HandleFunc("GET /api/v1/inspection-templates/{id}", h.handleGetTemplate)
	mux.HandleFunc("POST /api/v1/inspections", h.handleCreateInspection)
	mux.HandleFunc("GET /api/v1/inspections", h.handleListInspections)
	mux.HandleFunc("GET /api/v1/inspections/{id}", h.handleGetInspection)
	mux.HandleFunc("PUT /api/v1/inspections/{id}/items", h.handleUpsertInspectionItems)
	mux.HandleFunc("POST /api/v1/inspections/{id}/pdf", h.handleGenerateInspectionPDF)
	mux.HandleFunc("POST /api/v1/incidents", h.handleCreateIncident)
	mux.HandleFunc("GET /api/v1/incidents", h.handleListIncidents)
	mux.HandleFunc("GET /api/v1/incidents/{id}", h.handleGetIncident)
	mux.HandleFunc("PATCH /api/v1/incidents/{id}", h.handleUpdateIncident)
}

func (h *Handler) handleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	var cmd application.CreateInspectionTemplateCmd
	if err := httpx.DecodeJSON(r, &cmd); err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}

	dto, err := h.svc.CreateTemplate(r.Context(), cmd)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.Created(w, dto)
}

func (h *Handler) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListTemplates(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if list == nil {
		list = []application.InspectionTemplateDTO{}
	}
	httpx.OK(w, list)
}

func (h *Handler) handleGetTemplate(w http.ResponseWriter, r *http.Request) {
	dto, err := h.svc.GetTemplate(r.Context(), r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.OK(w, dto)
}

func (h *Handler) handleCreateInspection(w http.ResponseWriter, r *http.Request) {
	var cmd application.CreateInspectionCmd
	if err := httpx.DecodeJSON(r, &cmd); err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}

	dto, err := h.svc.CreateInspection(r.Context(), cmd)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.Created(w, dto)
}

func (h *Handler) handleListInspections(w http.ResponseWriter, r *http.Request) {
	leaseID := strings.TrimSpace(r.URL.Query().Get("lease_id"))
	if leaseID == "" {
		httpx.OK(w, []application.InspectionDTO{})
		return
	}
	list, err := h.svc.ListInspectionsByLease(r.Context(), leaseID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if list == nil {
		list = []application.InspectionDTO{}
	}
	httpx.OK(w, list)
}

func (h *Handler) handleGetInspection(w http.ResponseWriter, r *http.Request) {
	dto, err := h.svc.GetInspection(r.Context(), r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.OK(w, dto)
}

func (h *Handler) handleUpsertInspectionItems(w http.ResponseWriter, r *http.Request) {
	inspectionID := r.PathValue("id")
	rawBody, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		httpx.BadRequest(w, "failed to read request body")
		return
	}

	var items []application.InspectionItemInput
	trimmed := bytes.TrimSpace(rawBody)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &items); err != nil {
			httpx.BadRequest(w, "invalid inspection items array payload")
			return
		}
	} else {
		var wrapper struct {
			Items []application.InspectionItemInput `json:"items"`
		}
		if err := json.Unmarshal(trimmed, &wrapper); err != nil {
			httpx.BadRequest(w, "invalid inspection items payload")
			return
		}
		items = wrapper.Items
	}

	dto, err := h.svc.UpsertInspectionItems(r.Context(), inspectionID, items)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.OK(w, dto)
}

func (h *Handler) handleGenerateInspectionPDF(w http.ResponseWriter, r *http.Request) {
	inspectionID := r.PathValue("id")
	report, err := h.svc.GenerateInspectionReport(r.Context(), inspectionID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.Created(w, report)
}

func (h *Handler) handleCreateIncident(w http.ResponseWriter, r *http.Request) {
	var cmd application.CreateIncidentCmd
	if err := httpx.DecodeJSON(r, &cmd); err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}

	dto, err := h.svc.CreateIncident(r.Context(), cmd)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.Created(w, dto)
}

func (h *Handler) handleListIncidents(w http.ResponseWriter, r *http.Request) {
	propertyID := strings.TrimSpace(r.URL.Query().Get("property_id"))
	leaseID := strings.TrimSpace(r.URL.Query().Get("lease_id"))
	list, err := h.svc.ListIncidents(r.Context(), propertyID, leaseID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if list == nil {
		list = []application.IncidentDTO{}
	}
	httpx.OK(w, list)
}

func (h *Handler) handleGetIncident(w http.ResponseWriter, r *http.Request) {
	dto, err := h.svc.GetIncident(r.Context(), r.PathValue("id"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.OK(w, dto)
}

func (h *Handler) handleUpdateIncident(w http.ResponseWriter, r *http.Request) {
	var cmd application.UpdateIncidentCmd
	if err := httpx.DecodeJSON(r, &cmd); err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}
	cmd.IncidentID = r.PathValue("id")

	dto, err := h.svc.UpdateIncident(r.Context(), cmd)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.OK(w, dto)
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrValidation):
		httpx.UnprocessableEntity(w, err.Error())
	case errors.Is(err, domain.ErrTemplateNotFound),
		errors.Is(err, domain.ErrInspectionNotFound),
		errors.Is(err, domain.ErrIncidentNotFound):
		httpx.NotFound(w, err.Error())
	default:
		httpx.LogInternalError(w, "internal inspection error", err)
	}
}
