package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yannamsellem/gestion-immo/internal/modules/accounting/application"
	"github.com/yannamsellem/gestion-immo/internal/platform/mcpserver"
)

// RegisterTools registers the atomic accounting MCP tools (`preview_financial_statement`,
// `generate_annual_accounts_pdf`, and `export_fec`) on reg.
func RegisterTools(reg *mcpserver.Registry, svc *application.Service) {
	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "preview_financial_statement",
			Description: "Preview an entity's Bilan, Compte de Résultat, and Soldes Intermédiaires de Gestion (SIG) for a fiscal year, with optional disabled_line_ids to dynamically toggle/exclude specific lines and recalculate all cascading subtotals.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"legal_entity_id"},
				"properties": map[string]any{
					"legal_entity_id":   map[string]any{"type": "string", "description": "Legal entity ID (e.g. SCI, SARL)"},
					"entity_id":         map[string]any{"type": "string", "description": "Alias for legal_entity_id"},
					"fiscal_year":       map[string]any{"type": "integer", "description": "Fiscal year (e.g. 2024)"},
					"disabled_line_ids": map[string]any{"type": "array", "description": "Optional array of line_id or account_code strings to exclude from statement subtotals"},
				},
			},
		},
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var req struct {
				LegalEntityID   string   `json:"legal_entity_id"`
				EntityID        string   `json:"entity_id"`
				FiscalYear      int      `json:"fiscal_year"`
				DisabledLineIDs []string `json:"disabled_line_ids"`
			}
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, fmt.Errorf("invalid preview_financial_statement arguments: %w", err)
			}
			entityID := strings.TrimSpace(req.LegalEntityID)
			if entityID == "" {
				entityID = strings.TrimSpace(req.EntityID)
			}
			return svc.PreviewFinancialStatements(ctx, entityID, req.FiscalYear, req.DisabledLineIDs)
		},
	)

	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "generate_annual_accounts_pdf",
			Description: "Generate the complete Comptes Annuels & Liasse Fiscale PDF (Cabinet SGCAM layout, CERFA 2065, 2050-2059-F, 1330-CVAE, 2072, Détail Actif/Passif/Résultat, SIG) excluding disabled_line_ids, upload to R2, and return its download URL.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"legal_entity_id"},
				"properties": map[string]any{
					"legal_entity_id":   map[string]any{"type": "string", "description": "Legal entity ID"},
					"entity_id":         map[string]any{"type": "string", "description": "Alias for legal_entity_id"},
					"fiscal_year":       map[string]any{"type": "integer", "description": "Fiscal year (e.g. 2024)"},
					"disabled_line_ids": map[string]any{"type": "array", "description": "Optional array of line_id or account_code strings to exclude from the generated PDF"},
				},
			},
		},
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var req struct {
				LegalEntityID   string   `json:"legal_entity_id"`
				EntityID        string   `json:"entity_id"`
				FiscalYear      int      `json:"fiscal_year"`
				DisabledLineIDs []string `json:"disabled_line_ids"`
			}
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, fmt.Errorf("invalid generate_annual_accounts_pdf arguments: %w", err)
			}
			entityID := strings.TrimSpace(req.LegalEntityID)
			if entityID == "" {
				entityID = strings.TrimSpace(req.EntityID)
			}
			return svc.GenerateAnnualAccountsPDF(ctx, entityID, req.FiscalYear, req.DisabledLineIDs)
		},
	)

	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "export_fec",
			Description: "Export the normative 18-column DGFiP Fichier des Écritures Comptables (FEC) for a legal entity and fiscal year.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"legal_entity_id"},
				"properties": map[string]any{
					"legal_entity_id": map[string]any{"type": "string", "description": "Legal entity ID"},
					"entity_id":       map[string]any{"type": "string", "description": "Alias for legal_entity_id"},
					"fiscal_year":     map[string]any{"type": "integer", "description": "Fiscal year (e.g. 2024)"},
				},
			},
		},
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var req struct {
				LegalEntityID string `json:"legal_entity_id"`
				EntityID      string `json:"entity_id"`
				FiscalYear    int    `json:"fiscal_year"`
			}
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, fmt.Errorf("invalid export_fec arguments: %w", err)
			}
			entityID := strings.TrimSpace(req.LegalEntityID)
			if entityID == "" {
				entityID = strings.TrimSpace(req.EntityID)
			}
			fecBytes, err := svc.ExportEntityFEC(ctx, entityID, req.FiscalYear)
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"legal_entity_id": entityID,
				"fiscal_year":     req.FiscalYear,
				"fec_content":     string(fecBytes),
			}, nil
		},
	)
}
