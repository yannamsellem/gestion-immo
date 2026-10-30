package http

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

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

type createTenantInput struct {
	Body application.CreateTenantCmd
}

type tenantOutput struct {
	Body application.TenantDTO
}

type listTenantsInput struct {
	IncludeArchived string `query:"include_archived" required:"false"`
}

type tenantListOutput struct {
	Body []application.TenantDTO
}

type archiveTenantInput struct {
	ID string `path:"id"`
}

type createLeaseInput struct {
	Body application.CreateLeaseCmd
}

type leaseOutput struct {
	Body application.LeaseDTO
}

type listLeasesInput struct {
	LegalEntityID string `query:"legal_entity_id" required:"false"`
	EntityID      string `query:"entity_id" required:"false"`
}

type leaseListOutput struct {
	Body []application.LeaseDTO
}

// GenerateRentCallsBody represents the request body for generating monthly rent calls.
type GenerateRentCallsBody struct {
	PeriodMonth string `json:"period_month,omitempty"`
}

type generateRentCallsInput struct {
	ID          string `path:"id"`
	PeriodMonth string `query:"period_month" required:"false"`
	Body        GenerateRentCallsBody
}

type listEntityRentCallsInput struct {
	ID          string `path:"id"`
	PeriodMonth string `query:"period_month" required:"false"`
}

type rentCallListOutput struct {
	Body []application.RentCallDTO
}

// RegisterRoutes mounts all lease REST endpoints on the Go 1.22+ ServeMux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	h.RegisterHumaRoutes(httpx.NewHumaAPI(mux))
}

// RegisterHumaRoutes registers all lease REST endpoints onto a Huma API.
func (h *Handler) RegisterHumaRoutes(api huma.API) {
	bearerSecurity := []map[string][]string{{"bearerAuth": {}}}

	huma.Register(api, huma.Operation{
		OperationID:   "createTenant",
		Method:        http.MethodPost,
		Path:          "/api/v1/tenants",
		Summary:       "Create a tenant",
		DefaultStatus: http.StatusCreated,
		Security:      bearerSecurity,
		Tags:          []string{"Lease"},
	}, h.handleCreateTenant)

	huma.Register(api, huma.Operation{
		OperationID:   "listTenants",
		Method:        http.MethodGet,
		Path:          "/api/v1/tenants",
		Summary:       "List tenants",
		DefaultStatus: http.StatusOK,
		Security:      bearerSecurity,
		Tags:          []string{"Lease"},
	}, h.handleListTenants)

	huma.Register(api, huma.Operation{
		OperationID:   "archiveTenant",
		Method:        http.MethodPost,
		Path:          "/api/v1/tenants/{id}/archive",
		Summary:       "Archive a tenant",
		DefaultStatus: http.StatusNoContent,
		Security:      bearerSecurity,
		Tags:          []string{"Lease"},
	}, h.handleArchiveTenant)

	huma.Register(api, huma.Operation{
		OperationID:   "createLease",
		Method:        http.MethodPost,
		Path:          "/api/v1/leases",
		Summary:       "Create a residential or commercial lease",
		DefaultStatus: http.StatusCreated,
		Security:      bearerSecurity,
		Tags:          []string{"Lease"},
	}, h.handleCreateLease)

	huma.Register(api, huma.Operation{
		OperationID:   "listLeases",
		Method:        http.MethodGet,
		Path:          "/api/v1/leases",
		Summary:       "List leases optionally filtered by legal entity",
		DefaultStatus: http.StatusOK,
		Security:      bearerSecurity,
		Tags:          []string{"Lease"},
	}, h.handleListLeases)

	huma.Register(api, huma.Operation{
		OperationID:   "generateRentCalls",
		Method:        http.MethodPost,
		Path:          "/api/v1/entities/{id}/rent-calls/generate",
		Summary:       "Generate monthly rent calls for a legal entity",
		DefaultStatus: http.StatusCreated,
		Security:      bearerSecurity,
		Tags:          []string{"Lease"},
	}, h.handleGenerateRentCalls)

	huma.Register(api, huma.Operation{
		OperationID:   "listEntityRentCalls",
		Method:        http.MethodGet,
		Path:          "/api/v1/entities/{id}/rent-calls",
		Summary:       "List rent calls for a legal entity",
		DefaultStatus: http.StatusOK,
		Security:      bearerSecurity,
		Tags:          []string{"Lease"},
	}, h.handleListEntityRentCalls)
}

func (h *Handler) handleCreateTenant(ctx context.Context, in *createTenantInput) (*tenantOutput, error) {
	dto, err := h.svc.CreateTenant(ctx, in.Body)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &tenantOutput{Body: dto}, nil
}

func (h *Handler) handleListTenants(ctx context.Context, in *listTenantsInput) (*tenantListOutput, error) {
	includeArchived := strings.EqualFold(in.IncludeArchived, "true") || in.IncludeArchived == "1"

	tenants, err := h.svc.ListTenants(ctx, includeArchived)
	if err != nil {
		return nil, mapDomainError(err)
	}
	if tenants == nil {
		tenants = []application.TenantDTO{}
	}
	return &tenantListOutput{Body: tenants}, nil
}

func (h *Handler) handleArchiveTenant(ctx context.Context, in *archiveTenantInput) (*struct{}, error) {
	if err := h.svc.ArchiveTenant(ctx, in.ID); err != nil {
		return nil, mapDomainError(err)
	}
	return nil, nil
}

func (h *Handler) handleCreateLease(ctx context.Context, in *createLeaseInput) (*leaseOutput, error) {
	dto, err := h.svc.CreateLease(ctx, in.Body)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &leaseOutput{Body: dto}, nil
}

func (h *Handler) handleListLeases(ctx context.Context, in *listLeasesInput) (*leaseListOutput, error) {
	entityID := strings.TrimSpace(in.LegalEntityID)
	if entityID == "" {
		entityID = strings.TrimSpace(in.EntityID)
	}

	var (
		leases []application.LeaseDTO
		err    error
	)
	if entityID != "" {
		leases, err = h.svc.ListLeasesByEntity(ctx, entityID)
	} else {
		leases, err = h.svc.ListLeases(ctx)
	}
	if err != nil {
		return nil, mapDomainError(err)
	}
	if leases == nil {
		leases = []application.LeaseDTO{}
	}
	return &leaseListOutput{Body: leases}, nil
}

func (h *Handler) handleGenerateRentCalls(ctx context.Context, in *generateRentCallsInput) (*rentCallListOutput, error) {
	periodMonth := strings.TrimSpace(in.Body.PeriodMonth)
	if periodMonth == "" {
		periodMonth = strings.TrimSpace(in.PeriodMonth)
	}
	if periodMonth == "" {
		periodMonth = time.Now().UTC().Format("2006-01")
	}

	calls, err := h.svc.GenerateMonthlyRentCalls(ctx, in.ID, periodMonth)
	if err != nil {
		return nil, mapDomainError(err)
	}
	if calls == nil {
		calls = []application.RentCallDTO{}
	}
	return &rentCallListOutput{Body: calls}, nil
}

func (h *Handler) handleListEntityRentCalls(ctx context.Context, in *listEntityRentCallsInput) (*rentCallListOutput, error) {
	calls, err := h.svc.ListRentCallsByEntity(ctx, in.ID)
	if err != nil {
		return nil, mapDomainError(err)
	}
	periodMonth := strings.TrimSpace(in.PeriodMonth)
	if periodMonth != "" {
		filtered := make([]application.RentCallDTO, 0, len(calls))
		for _, c := range calls {
			if c.PeriodMonth == periodMonth {
				filtered = append(filtered, c)
			}
		}
		calls = filtered
	}
	if calls == nil {
		calls = []application.RentCallDTO{}
	}
	return &rentCallListOutput{Body: calls}, nil
}

func mapDomainError(err error) error {
	switch {
	case errors.Is(err, domain.ErrValidation):
		return httpx.StatusError(http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, domain.ErrDuplicatePeriod):
		return httpx.StatusError(http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrTenantNotFound),
		errors.Is(err, domain.ErrLeaseNotFound),
		errors.Is(err, domain.ErrRentCallNotFound):
		return httpx.StatusError(http.StatusNotFound, err.Error())
	default:
		return httpx.StatusError(http.StatusInternalServerError, "internal lease error")
	}
}
