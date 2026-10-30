package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"

	"github.com/danielgtaylor/huma/v2"

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

type createInspectionTemplateInput struct {
	Body application.CreateInspectionTemplateCmd
}

type inspectionTemplateOutput struct {
	Body application.InspectionTemplateDTO
}

type inspectionTemplateListOutput struct {
	Body []application.InspectionTemplateDTO
}

type idInput struct {
	ID string `path:"id"`
}

type createInspectionInput struct {
	Body application.CreateInspectionCmd
}

type listInspectionsInput struct {
	LeaseID string `query:"lease_id" required:"false"`
}

type inspectionOutput struct {
	Body application.InspectionDTO
}

type inspectionListOutput struct {
	Body []application.InspectionDTO
}

// UpsertInspectionItemsBody represents the request body for upserting inspection items.
type UpsertInspectionItemsBody struct {
	Items []application.InspectionItemInput `json:"items"`
}

type upsertInspectionItemsInput struct {
	ID      string `path:"id"`
	RawBody []byte
}

type inspectionReportOutput struct {
	Body application.InspectionReportDTO
}

type createIncidentInput struct {
	Body application.CreateIncidentCmd
}

type listIncidentsInput struct {
	PropertyID string `query:"property_id" required:"false"`
	LeaseID    string `query:"lease_id" required:"false"`
}

type updateIncidentInput struct {
	ID   string `path:"id"`
	Body application.UpdateIncidentCmd
}

type incidentOutput struct {
	Body application.IncidentDTO
}

type incidentListOutput struct {
	Body []application.IncidentDTO
}

// RegisterRoutes mounts all inspection and incident REST endpoints on the Go 1.22+ ServeMux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	h.RegisterHumaRoutes(httpx.NewHumaAPI(mux))
}

// RegisterHumaRoutes registers all inspection and incident REST endpoints onto a Huma API.
func (h *Handler) RegisterHumaRoutes(api huma.API) {
	bearerSecurity := []map[string][]string{{"bearerAuth": {}}}

	huma.Register(api, huma.Operation{
		OperationID:   "createInspectionTemplate",
		Method:        http.MethodPost,
		Path:          "/api/v1/inspection-templates",
		Summary:       "Create a reusable room/element inspection template",
		DefaultStatus: http.StatusCreated,
		Security:      bearerSecurity,
		Tags:          []string{"Inspection"},
	}, h.handleCreateTemplate)

	huma.Register(api, huma.Operation{
		OperationID:   "listInspectionTemplates",
		Method:        http.MethodGet,
		Path:          "/api/v1/inspection-templates",
		Summary:       "List inspection templates",
		DefaultStatus: http.StatusOK,
		Security:      bearerSecurity,
		Tags:          []string{"Inspection"},
	}, h.handleListTemplates)

	huma.Register(api, huma.Operation{
		OperationID:   "getInspectionTemplate",
		Method:        http.MethodGet,
		Path:          "/api/v1/inspection-templates/{id}",
		Summary:       "Get an inspection template by ID",
		DefaultStatus: http.StatusOK,
		Security:      bearerSecurity,
		Tags:          []string{"Inspection"},
	}, h.handleGetTemplate)

	huma.Register(api, huma.Operation{
		OperationID:   "createInspection",
		Method:        http.MethodPost,
		Path:          "/api/v1/inspections",
		Summary:       "Create a move-in or move-out inspection",
		DefaultStatus: http.StatusCreated,
		Security:      bearerSecurity,
		Tags:          []string{"Inspection"},
	}, h.handleCreateInspection)

	huma.Register(api, huma.Operation{
		OperationID:   "listInspections",
		Method:        http.MethodGet,
		Path:          "/api/v1/inspections",
		Summary:       "List inspections for a lease",
		DefaultStatus: http.StatusOK,
		Security:      bearerSecurity,
		Tags:          []string{"Inspection"},
	}, h.handleListInspections)

	huma.Register(api, huma.Operation{
		OperationID:   "getInspection",
		Method:        http.MethodGet,
		Path:          "/api/v1/inspections/{id}",
		Summary:       "Get an inspection by ID",
		DefaultStatus: http.StatusOK,
		Security:      bearerSecurity,
		Tags:          []string{"Inspection"},
	}, h.handleGetInspection)

	upsertItemsSchema := huma.SchemaFromType(api.OpenAPI().Components.Schemas, reflect.TypeFor[UpsertInspectionItemsBody]())
	huma.Register(api, huma.Operation{
		OperationID:      "upsertInspectionItems",
		Method:           http.MethodPut,
		Path:             "/api/v1/inspections/{id}/items",
		Summary:          "Upsert room/element condition ratings on an inspection",
		DefaultStatus:    http.StatusOK,
		MaxBodyBytes:     4 << 20,
		SkipValidateBody: true,
		Security:         bearerSecurity,
		Tags:             []string{"Inspection"},
	}, h.handleUpsertInspectionItems)
	if pathItem := api.OpenAPI().Paths["/api/v1/inspections/{id}/items"]; pathItem != nil && pathItem.Put != nil {
		pathItem.Put.RequestBody = &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {Schema: upsertItemsSchema},
			},
		}
	}

	huma.Register(api, huma.Operation{
		OperationID:   "generateInspectionPdf",
		Method:        http.MethodPost,
		Path:          "/api/v1/inspections/{id}/pdf",
		Summary:       "Generate a PDF inspection report (standalone or move-in vs move-out comparison)",
		DefaultStatus: http.StatusCreated,
		Security:      bearerSecurity,
		Tags:          []string{"Inspection"},
	}, h.handleGenerateInspectionPDF)

	huma.Register(api, huma.Operation{
		OperationID:   "createIncident",
		Method:        http.MethodPost,
		Path:          "/api/v1/incidents",
		Summary:       "Open a property, lot, or lease incident",
		DefaultStatus: http.StatusCreated,
		Security:      bearerSecurity,
		Tags:          []string{"Inspection"},
	}, h.handleCreateIncident)

	huma.Register(api, huma.Operation{
		OperationID:   "listIncidents",
		Method:        http.MethodGet,
		Path:          "/api/v1/incidents",
		Summary:       "List incidents filtered by property or lease",
		DefaultStatus: http.StatusOK,
		Security:      bearerSecurity,
		Tags:          []string{"Inspection"},
	}, h.handleListIncidents)

	huma.Register(api, huma.Operation{
		OperationID:   "getIncident",
		Method:        http.MethodGet,
		Path:          "/api/v1/incidents/{id}",
		Summary:       "Get an incident by ID",
		DefaultStatus: http.StatusOK,
		Security:      bearerSecurity,
		Tags:          []string{"Inspection"},
	}, h.handleGetIncident)

	huma.Register(api, huma.Operation{
		OperationID:   "updateIncident",
		Method:        http.MethodPatch,
		Path:          "/api/v1/incidents/{id}",
		Summary:       "Update an incident status, insurance claim number, cost, observations, or photos",
		DefaultStatus: http.StatusOK,
		Security:      bearerSecurity,
		Tags:          []string{"Inspection"},
	}, h.handleUpdateIncident)
}

func (h *Handler) handleCreateTemplate(ctx context.Context, in *createInspectionTemplateInput) (*inspectionTemplateOutput, error) {
	dto, err := h.svc.CreateTemplate(ctx, in.Body)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &inspectionTemplateOutput{Body: dto}, nil
}

func (h *Handler) handleListTemplates(ctx context.Context, _ *struct{}) (*inspectionTemplateListOutput, error) {
	list, err := h.svc.ListTemplates(ctx)
	if err != nil {
		return nil, mapDomainError(err)
	}
	if list == nil {
		list = []application.InspectionTemplateDTO{}
	}
	return &inspectionTemplateListOutput{Body: list}, nil
}

func (h *Handler) handleGetTemplate(ctx context.Context, in *idInput) (*inspectionTemplateOutput, error) {
	dto, err := h.svc.GetTemplate(ctx, in.ID)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &inspectionTemplateOutput{Body: dto}, nil
}

func (h *Handler) handleCreateInspection(ctx context.Context, in *createInspectionInput) (*inspectionOutput, error) {
	dto, err := h.svc.CreateInspection(ctx, in.Body)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &inspectionOutput{Body: dto}, nil
}

func (h *Handler) handleListInspections(ctx context.Context, in *listInspectionsInput) (*inspectionListOutput, error) {
	leaseID := strings.TrimSpace(in.LeaseID)
	if leaseID == "" {
		return &inspectionListOutput{Body: []application.InspectionDTO{}}, nil
	}
	list, err := h.svc.ListInspectionsByLease(ctx, leaseID)
	if err != nil {
		return nil, mapDomainError(err)
	}
	if list == nil {
		list = []application.InspectionDTO{}
	}
	return &inspectionListOutput{Body: list}, nil
}

func (h *Handler) handleGetInspection(ctx context.Context, in *idInput) (*inspectionOutput, error) {
	dto, err := h.svc.GetInspection(ctx, in.ID)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &inspectionOutput{Body: dto}, nil
}

func (h *Handler) handleUpsertInspectionItems(ctx context.Context, in *upsertInspectionItemsInput) (*inspectionOutput, error) {
	var items []application.InspectionItemInput
	trimmed := bytes.TrimSpace(in.RawBody)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &items); err != nil {
			return nil, httpx.StatusError(http.StatusBadRequest, "invalid inspection items array payload")
		}
	} else {
		var wrapper UpsertInspectionItemsBody
		if err := json.Unmarshal(trimmed, &wrapper); err != nil {
			return nil, httpx.StatusError(http.StatusBadRequest, "invalid inspection items payload")
		}
		items = wrapper.Items
	}

	dto, err := h.svc.UpsertInspectionItems(ctx, in.ID, items)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &inspectionOutput{Body: dto}, nil
}

func (h *Handler) handleGenerateInspectionPDF(ctx context.Context, in *idInput) (*inspectionReportOutput, error) {
	report, err := h.svc.GenerateInspectionReport(ctx, in.ID)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &inspectionReportOutput{Body: report}, nil
}

func (h *Handler) handleCreateIncident(ctx context.Context, in *createIncidentInput) (*incidentOutput, error) {
	dto, err := h.svc.CreateIncident(ctx, in.Body)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &incidentOutput{Body: dto}, nil
}

func (h *Handler) handleListIncidents(ctx context.Context, in *listIncidentsInput) (*incidentListOutput, error) {
	list, err := h.svc.ListIncidents(ctx, strings.TrimSpace(in.PropertyID), strings.TrimSpace(in.LeaseID))
	if err != nil {
		return nil, mapDomainError(err)
	}
	if list == nil {
		list = []application.IncidentDTO{}
	}
	return &incidentListOutput{Body: list}, nil
}

func (h *Handler) handleGetIncident(ctx context.Context, in *idInput) (*incidentOutput, error) {
	dto, err := h.svc.GetIncident(ctx, in.ID)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &incidentOutput{Body: dto}, nil
}

func (h *Handler) handleUpdateIncident(ctx context.Context, in *updateIncidentInput) (*incidentOutput, error) {
	cmd := in.Body
	cmd.IncidentID = in.ID

	dto, err := h.svc.UpdateIncident(ctx, cmd)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &incidentOutput{Body: dto}, nil
}

func mapDomainError(err error) error {
	switch {
	case errors.Is(err, domain.ErrValidation):
		return httpx.StatusError(http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, domain.ErrTemplateNotFound),
		errors.Is(err, domain.ErrInspectionNotFound),
		errors.Is(err, domain.ErrIncidentNotFound):
		return httpx.StatusError(http.StatusNotFound, err.Error())
	default:
		return httpx.StatusError(http.StatusInternalServerError, "internal inspection error")
	}
}
