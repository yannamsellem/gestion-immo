package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yannamsellem/gestion-immo/internal/modules/portfolio/application"
	"github.com/yannamsellem/gestion-immo/internal/platform/mcpserver"
)

// RegisterTools registers all atomic portfolio MCP tools on reg.
func RegisterTools(reg *mcpserver.Registry, svc *application.Service) {
	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "create_legal_entity",
			Description: "Create a portfolio legal entity (SCI_IR, SCI_IS, SARL_IS, FAMILY_SARL, SAS, SOLE_PROPRIETORSHIP, JOINT_OWNERSHIP).",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"name", "entity_type"},
				"properties": map[string]any{
					"name":                map[string]any{"type": "string", "description": "Legal entity name (e.g., SARL INVESTISSIMO)"},
					"entity_type":         map[string]any{"type": "string", "enum": []string{"SCI_IR", "SCI_IS", "SARL_IS", "FAMILY_SARL", "SAS", "SOLE_PROPRIETORSHIP", "JOINT_OWNERSHIP"}},
					"siret":               map[string]any{"type": "string", "description": "14-digit SIRET identifier"},
					"naf_code":            map[string]any{"type": "string", "description": "INSEE NAF/APE code (e.g., 6810Z, 6820B)"},
					"address":             map[string]any{"type": "string", "description": "Registered office address"},
					"share_capital_cents": map[string]any{"type": "integer", "description": "Share capital in euro cents"},
					"total_shares":        map[string]any{"type": "integer", "description": "Total number of shares issued by the entity"},
				},
			},
		},
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var cmd application.CreateLegalEntityCmd
			if err := json.Unmarshal(args, &cmd); err != nil {
				return nil, fmt.Errorf("invalid create_legal_entity arguments: %w", err)
			}
			return svc.CreateLegalEntity(ctx, cmd)
		},
	)

	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "upsert_shareholder",
			Description: "Create or update an associate/shareholder on a legal entity with exact CERFA 2059-F detention basis points and current account balance.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"legal_entity_id", "full_name", "shares_count"},
				"properties": map[string]any{
					"legal_entity_id":         map[string]any{"type": "string", "description": "Target legal entity ID"},
					"full_name":               map[string]any{"type": "string", "description": "Full name of the shareholder"},
					"shares_count":            map[string]any{"type": "integer", "description": "Number of shares held"},
					"current_account_cents":   map[string]any{"type": "integer", "description": "Compte courant d'associé (PCG 455) balance in euro cents"},
					"remuneration_paid_cents": map[string]any{"type": "integer", "description": "Annual remuneration paid in euro cents"},
				},
			},
		},
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var cmd application.UpsertShareholderCmd
			if err := json.Unmarshal(args, &cmd); err != nil {
				return nil, fmt.Errorf("invalid upsert_shareholder arguments: %w", err)
			}
			return svc.UpsertShareholder(ctx, cmd)
		},
	)

	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "list_entities",
			Description: "List all portfolio legal entities.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		func(ctx context.Context, _ json.RawMessage) (any, error) {
			return svc.ListLegalEntities(ctx)
		},
	)

	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "get_entity_summary",
			Description: "Get full portfolio summary for a legal entity including CERFA 2059-F shareholders, properties, lots, and total book value.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"entity_id"},
				"properties": map[string]any{
					"entity_id": map[string]any{"type": "string", "description": "Legal entity ID"},
				},
			},
		},
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var req struct {
				EntityID      string `json:"entity_id"`
				LegalEntityID string `json:"legal_entity_id"`
			}
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, fmt.Errorf("invalid get_entity_summary arguments: %w", err)
			}
			id := req.EntityID
			if id == "" {
				id = req.LegalEntityID
			}
			return svc.GetEntityPortfolioSummary(ctx, id)
		},
	)

	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "create_property",
			Description: "Create a real estate property classified as FIXED_ASSET, MERCHANDISE, or FINISHED_GOOD with acquisition cost and work stock.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"legal_entity_id", "label"},
				"properties": map[string]any{
					"legal_entity_id":     map[string]any{"type": "string", "description": "Owning legal entity ID"},
					"label":               map[string]any{"type": "string", "description": "Property label"},
					"property_type":       map[string]any{"type": "string", "description": "Property type (e.g., BUILDING, APARTMENT, COMMERCIAL_PREMISES)"},
					"accounting_category": map[string]any{"type": "string", "enum": []string{"FIXED_ASSET", "MERCHANDISE", "FINISHED_GOOD"}},
					"address":             map[string]any{"type": "string", "description": "Property street address"},
					"acquisition_cents":   map[string]any{"type": "integer", "description": "Acquisition price in euro cents"},
					"work_stock_cents":    map[string]any{"type": "integer", "description": "Capitalized renovation work stock in euro cents"},
				},
			},
		},
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var cmd application.CreatePropertyCmd
			if err := json.Unmarshal(args, &cmd); err != nil {
				return nil, fmt.Errorf("invalid create_property arguments: %w", err)
			}
			return svc.CreateProperty(ctx, cmd)
		},
	)

	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "create_lot",
			Description: "Create a rentable lot (apartment, commercial shop, parking, cellar) inside a property.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"property_id", "label"},
				"properties": map[string]any{
					"property_id":     map[string]any{"type": "string", "description": "Parent property ID"},
					"label":           map[string]any{"type": "string", "description": "Lot label (e.g., Lot 1 - Ground Floor Commercial)"},
					"lot_type":        map[string]any{"type": "string", "description": "Lot type (APARTMENT, COMMERCIAL_PREMISES, PARKING, CELLAR)"},
					"surface_m2":      map[string]any{"type": "integer", "description": "Surface area in square meters"},
					"ownership_share": map[string]any{"type": "integer", "description": "Co-ownership share (thousandths)"},
					"floor":           map[string]any{"type": "string", "description": "Floor identifier"},
				},
			},
		},
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var cmd application.CreateLotCmd
			if err := json.Unmarshal(args, &cmd); err != nil {
				return nil, fmt.Errorf("invalid create_lot arguments: %w", err)
			}
			return svc.CreateLot(ctx, cmd)
		},
	)

	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "list_properties",
			Description: "List all properties and their lots for a given legal entity.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"legal_entity_id"},
				"properties": map[string]any{
					"legal_entity_id": map[string]any{"type": "string", "description": "Legal entity ID"},
				},
			},
		},
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var req struct {
				LegalEntityID string `json:"legal_entity_id"`
				EntityID      string `json:"entity_id"`
			}
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, fmt.Errorf("invalid list_properties arguments: %w", err)
			}
			id := req.LegalEntityID
			if id == "" {
				id = req.EntityID
			}
			return svc.ListProperties(ctx, id)
		},
	)
}
