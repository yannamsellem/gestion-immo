package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

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

type createEntityInput struct {
	Body application.CreateLegalEntityCmd
}

type entityOutput struct {
	Body application.LegalEntityDTO
}

type entityListOutput struct {
	Body []application.LegalEntityDTO
}

type entityIDInput struct {
	ID string `path:"id"`
}

type entitySummaryOutput struct {
	Body application.EntityPortfolioSummaryDTO
}

type upsertShareholderInput struct {
	ID   string `path:"id"`
	Body application.UpsertShareholderCmd
}

type shareholderOutput struct {
	Body application.ShareholderDTO
}

type createPropertyInput struct {
	ID   string `path:"id"`
	Body application.CreatePropertyCmd
}

type propertyOutput struct {
	Body application.PropertyDTO
}

type propertyListOutput struct {
	Body []application.PropertyDTO
}

type createLotInput struct {
	ID   string `path:"id"`
	Body application.CreateLotCmd
}

type lotOutput struct {
	Body application.LotDTO
}

// RegisterRoutes mounts all portfolio REST endpoints on the Go 1.22+ ServeMux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	h.RegisterHumaRoutes(httpx.NewHumaAPI(mux))
}

// RegisterHumaRoutes registers all portfolio REST endpoints onto a Huma API.
func (h *Handler) RegisterHumaRoutes(api huma.API) {
	bearerSecurity := []map[string][]string{{"bearerAuth": {}}}

	huma.Register(api, huma.Operation{
		OperationID:   "createEntity",
		Method:        http.MethodPost,
		Path:          "/api/v1/entities",
		Summary:       "Create a legal entity (SCI, SARL, LMNP, LMP, own name)",
		DefaultStatus: http.StatusCreated,
		Security:      bearerSecurity,
		Tags:          []string{"Portfolio"},
	}, h.handleCreateEntity)

	huma.Register(api, huma.Operation{
		OperationID:   "listEntities",
		Method:        http.MethodGet,
		Path:          "/api/v1/entities",
		Summary:       "List all legal entities",
		DefaultStatus: http.StatusOK,
		Security:      bearerSecurity,
		Tags:          []string{"Portfolio"},
	}, h.handleListEntities)

	huma.Register(api, huma.Operation{
		OperationID:   "getEntitySummary",
		Method:        http.MethodGet,
		Path:          "/api/v1/entities/{id}",
		Summary:       "Get a legal entity summary with shareholders, properties, and lots",
		DefaultStatus: http.StatusOK,
		Security:      bearerSecurity,
		Tags:          []string{"Portfolio"},
	}, h.handleGetEntitySummary)

	huma.Register(api, huma.Operation{
		OperationID:   "upsertShareholder",
		Method:        http.MethodPost,
		Path:          "/api/v1/entities/{id}/shareholders",
		Summary:       "Create or update a shareholder on a legal entity",
		DefaultStatus: http.StatusOK,
		Security:      bearerSecurity,
		Tags:          []string{"Portfolio"},
	}, h.handleUpsertShareholder)

	huma.Register(api, huma.Operation{
		OperationID:   "createProperty",
		Method:        http.MethodPost,
		Path:          "/api/v1/entities/{id}/properties",
		Summary:       "Create a real estate property under a legal entity",
		DefaultStatus: http.StatusCreated,
		Security:      bearerSecurity,
		Tags:          []string{"Portfolio"},
	}, h.handleCreateProperty)

	huma.Register(api, huma.Operation{
		OperationID:   "listProperties",
		Method:        http.MethodGet,
		Path:          "/api/v1/entities/{id}/properties",
		Summary:       "List properties and lots belonging to a legal entity",
		DefaultStatus: http.StatusOK,
		Security:      bearerSecurity,
		Tags:          []string{"Portfolio"},
	}, h.handleListProperties)

	huma.Register(api, huma.Operation{
		OperationID:   "createLot",
		Method:        http.MethodPost,
		Path:          "/api/v1/properties/{id}/lots",
		Summary:       "Create a rentable lot within a property",
		DefaultStatus: http.StatusCreated,
		Security:      bearerSecurity,
		Tags:          []string{"Portfolio"},
	}, h.handleCreateLot)
}

func (h *Handler) handleCreateEntity(ctx context.Context, in *createEntityInput) (*entityOutput, error) {
	dto, err := h.svc.CreateLegalEntity(ctx, in.Body)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &entityOutput{Body: dto}, nil
}

func (h *Handler) handleListEntities(ctx context.Context, _ *struct{}) (*entityListOutput, error) {
	entities, err := h.svc.ListLegalEntities(ctx)
	if err != nil {
		return nil, mapDomainError(err)
	}
	if entities == nil {
		entities = []application.LegalEntityDTO{}
	}
	return &entityListOutput{Body: entities}, nil
}

func (h *Handler) handleGetEntitySummary(ctx context.Context, in *entityIDInput) (*entitySummaryOutput, error) {
	summary, err := h.svc.GetEntityPortfolioSummary(ctx, in.ID)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &entitySummaryOutput{Body: summary}, nil
}

func (h *Handler) handleUpsertShareholder(ctx context.Context, in *upsertShareholderInput) (*shareholderOutput, error) {
	cmd := in.Body
	cmd.LegalEntityID = in.ID
	dto, err := h.svc.UpsertShareholder(ctx, cmd)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &shareholderOutput{Body: dto}, nil
}

func (h *Handler) handleCreateProperty(ctx context.Context, in *createPropertyInput) (*propertyOutput, error) {
	cmd := in.Body
	cmd.LegalEntityID = in.ID
	dto, err := h.svc.CreateProperty(ctx, cmd)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &propertyOutput{Body: dto}, nil
}

func (h *Handler) handleListProperties(ctx context.Context, in *entityIDInput) (*propertyListOutput, error) {
	props, err := h.svc.ListProperties(ctx, in.ID)
	if err != nil {
		return nil, mapDomainError(err)
	}
	if props == nil {
		props = []application.PropertyDTO{}
	}
	return &propertyListOutput{Body: props}, nil
}

func (h *Handler) handleCreateLot(ctx context.Context, in *createLotInput) (*lotOutput, error) {
	cmd := in.Body
	cmd.PropertyID = in.ID
	dto, err := h.svc.CreateLot(ctx, cmd)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &lotOutput{Body: dto}, nil
}

func mapDomainError(err error) error {
	switch {
	case errors.Is(err, domain.ErrValidation):
		return httpx.StatusError(http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, domain.ErrDuplicateSIRET):
		return httpx.StatusError(http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrLegalEntityNotFound),
		errors.Is(err, domain.ErrPropertyNotFound),
		errors.Is(err, domain.ErrLotNotFound),
		errors.Is(err, domain.ErrShareholderNotFound):
		return httpx.StatusError(http.StatusNotFound, err.Error())
	default:
		return httpx.StatusError(http.StatusInternalServerError, "internal portfolio error")
	}
}
