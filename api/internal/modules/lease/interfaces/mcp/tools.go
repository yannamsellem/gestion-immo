package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yannamsellem/gestion-immo/internal/modules/lease/application"
	"github.com/yannamsellem/gestion-immo/internal/platform/mcpserver"
)

// RegisterTools registers all atomic lease MCP tools on reg.
func RegisterTools(reg *mcpserver.Registry, svc *application.Service) {
	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "create_tenant",
			Description: "Create a present tenant.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"full_name"},
				"properties": map[string]any{
					"full_name": map[string]any{"type": "string", "description": "Tenant full name or company name"},
					"email":     map[string]any{"type": "string", "description": "Contact email address"},
					"phone":     map[string]any{"type": "string", "description": "Contact phone number"},
					"address":   map[string]any{"type": "string", "description": "Postal address"},
				},
			},
		},
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var cmd application.CreateTenantCmd
			if err := json.Unmarshal(args, &cmd); err != nil {
				return nil, fmt.Errorf("invalid create_tenant arguments: %w", err)
			}
			return svc.CreateTenant(ctx, cmd)
		},
	)

	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "create_lease",
			Description: "Create a residential or commercial lease with optional commercial VAT (basis points) and GLI policy metadata.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"legal_entity_id", "lot_id", "tenant_id", "lease_type", "rent_net_cents"},
				"properties": map[string]any{
					"legal_entity_id":        map[string]any{"type": "string", "description": "Owning legal entity ID"},
					"lot_id":                 map[string]any{"type": "string", "description": "Rented lot ID"},
					"tenant_id":              map[string]any{"type": "string", "description": "Tenant ID"},
					"lease_type":             map[string]any{"type": "string", "enum": []string{"RESIDENTIAL_UNFURNISHED", "RESIDENTIAL_FURNISHED", "COMMERCIAL_3_6_9", "PROFESSIONAL", "PARKING", "SHORT_TERM"}},
					"rent_net_cents":         map[string]any{"type": "integer", "description": "Monthly net rent (excl. VAT) in euro cents"},
					"charges_cents":          map[string]any{"type": "integer", "description": "Monthly service charges provision in euro cents"},
					"subject_to_vat":         map[string]any{"type": "boolean", "description": "Whether rent is subject to commercial VAT"},
					"vat_rate_basis_points":  map[string]any{"type": "integer", "description": "VAT rate in basis points (e.g., 2000 for 20.00%)"},
					"security_deposit_cents": map[string]any{"type": "integer", "description": "Security deposit in euro cents"},
					"due_day_of_month":       map[string]any{"type": "integer", "description": "Payment due day of month (1-31)"},
					"gli_policy_number":      map[string]any{"type": "string", "description": "Unpaid rent insurance (GLI) policy reference"},
					"guarantor_name":         map[string]any{"type": "string", "description": "Guarantor full name"},
					"start_date":             map[string]any{"type": "string", "description": "Lease start date (YYYY-MM-DD)"},
				},
			},
		},
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var cmd application.CreateLeaseCmd
			if err := json.Unmarshal(args, &cmd); err != nil {
				return nil, fmt.Errorf("invalid create_lease arguments: %w", err)
			}
			return svc.CreateLease(ctx, cmd)
		},
	)

	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "list_leases",
			Description: "List all leases, optionally filtered by legal_entity_id.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"legal_entity_id": map[string]any{"type": "string", "description": "Optional legal entity ID filter"},
				},
			},
		},
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var req struct {
				LegalEntityID string `json:"legal_entity_id"`
				EntityID      string `json:"entity_id"`
			}
			if len(args) > 0 {
				if err := json.Unmarshal(args, &req); err != nil {
					return nil, fmt.Errorf("invalid list_leases arguments: %w", err)
				}
			}
			id := strings.TrimSpace(req.LegalEntityID)
			if id == "" {
				id = strings.TrimSpace(req.EntityID)
			}
			if id != "" {
				return svc.ListLeasesByEntity(ctx, id)
			}
			return svc.ListLeases(ctx)
		},
	)

	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "generate_monthly_rent_calls",
			Description: "Generate monthly rent calls for all active leases belonging to a legal entity for a given period_month (YYYY-MM).",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"legal_entity_id", "period_month"},
				"properties": map[string]any{
					"legal_entity_id": map[string]any{"type": "string", "description": "Legal entity ID"},
					"entity_id":       map[string]any{"type": "string", "description": "Alias for legal_entity_id"},
					"period_month":    map[string]any{"type": "string", "description": "Billing period in YYYY-MM format (e.g. 2026-03)"},
				},
			},
		},
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var req struct {
				LegalEntityID string `json:"legal_entity_id"`
				EntityID      string `json:"entity_id"`
				PeriodMonth   string `json:"period_month"`
			}
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, fmt.Errorf("invalid generate_monthly_rent_calls arguments: %w", err)
			}
			entityID := strings.TrimSpace(req.LegalEntityID)
			if entityID == "" {
				entityID = strings.TrimSpace(req.EntityID)
			}
			return svc.GenerateMonthlyRentCalls(ctx, entityID, req.PeriodMonth)
		},
	)
}
