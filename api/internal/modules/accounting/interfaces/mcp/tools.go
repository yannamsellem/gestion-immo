package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yannamsellem/gestion-immo/internal/modules/accounting/application"
	"github.com/yannamsellem/gestion-immo/internal/platform/mcpserver"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

// RegisterTools registers all atomic accounting MCP tools on reg.
func RegisterTools(reg *mcpserver.Registry, svc *application.Service) {
	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "record_journal_entry",
			Description: "Record a balanced double-entry PCG journal entry for a legal entity and synchronize the general ledger balance lines.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"legal_entity_id", "label", "lines"},
				"properties": map[string]any{
					"legal_entity_id": map[string]any{"type": "string", "description": "Legal entity ID"},
					"entity_id":       map[string]any{"type": "string", "description": "Alias for legal_entity_id"},
					"fiscal_year":     map[string]any{"type": "integer", "description": "Optional fiscal year"},
					"journal_code":    map[string]any{"type": "string", "description": "Journal code (e.g., VE, AC, BQ, CA, OD)"},
					"journal_label":   map[string]any{"type": "string", "description": "Optional journal label"},
					"entry_number":    map[string]any{"type": "string", "description": "Optional entry number"},
					"entry_date":      map[string]any{"type": "string", "description": "Entry date (YYYY-MM-DD)"},
					"document_ref":    map[string]any{"type": "string", "description": "Supporting document reference"},
					"document_date":   map[string]any{"type": "string", "description": "Supporting document date (YYYY-MM-DD)"},
					"label":           map[string]any{"type": "string", "description": "Journal entry label"},
					"lines":           map[string]any{"type": "array", "description": "Balanced debit/credit lines ({account_code, account_label, sub_account_code, label, debit_cents, credit_cents})"},
				},
			},
		},
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var req struct {
				application.RecordJournalEntryCmd
				EntityID string `json:"entity_id"`
			}
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, fmt.Errorf("invalid record_journal_entry arguments: %w", err)
			}
			if strings.TrimSpace(req.LegalEntityID) == "" {
				req.LegalEntityID = strings.TrimSpace(req.EntityID)
			}
			return svc.RecordJournalEntry(ctx, req.RecordJournalEntryCmd)
		},
	)

	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "record_expense_invoice",
			Description: "Record a supplier expense invoice in the AC (Purchases) journal by debiting expense_account_code + 445660 (deductible VAT) and crediting 401000 (Suppliers).",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"legal_entity_id", "amount_ht_cents"},
				"properties": map[string]any{
					"legal_entity_id":      map[string]any{"type": "string", "description": "Legal entity ID"},
					"entity_id":            map[string]any{"type": "string", "description": "Alias for legal_entity_id"},
					"supplier_name":        map[string]any{"type": "string", "description": "Supplier name (used as sub-account on 401000)"},
					"expense_account_code": map[string]any{"type": "string", "description": "PCG Class 6 expense account code (e.g. 606400, 614000, 615200)"},
					"label":                map[string]any{"type": "string", "description": "Invoice description label"},
					"invoice_ref":          map[string]any{"type": "string", "description": "Supplier invoice reference"},
					"invoice_date":         map[string]any{"type": "string", "description": "Invoice date (YYYY-MM-DD)"},
					"amount_ht_cents":      map[string]any{"type": "integer", "description": "Amount excluding VAT in euro cents"},
					"vat_cents":            map[string]any{"type": "integer", "description": "Deductible VAT (PCG 445660) in euro cents"},
				},
			},
		},
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var req struct {
				LegalEntityID      string            `json:"legal_entity_id"`
				EntityID           string            `json:"entity_id"`
				SupplierName       string            `json:"supplier_name"`
				ExpenseAccountCode string            `json:"expense_account_code"`
				Label              string            `json:"label"`
				InvoiceRef         string            `json:"invoice_ref"`
				InvoiceDate        string            `json:"invoice_date"`
				AmountHTCents      money.AmountCents `json:"amount_ht_cents"`
				VATCents           money.AmountCents `json:"vat_cents"`
			}
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, fmt.Errorf("invalid record_expense_invoice arguments: %w", err)
			}
			entityID := strings.TrimSpace(req.LegalEntityID)
			if entityID == "" {
				entityID = strings.TrimSpace(req.EntityID)
			}
			return svc.RecordExpenseInvoice(ctx, application.RecordExpenseInvoiceCmd{
				LegalEntityID:      entityID,
				SupplierName:       req.SupplierName,
				ExpenseAccountCode: req.ExpenseAccountCode,
				Label:              req.Label,
				InvoiceRef:         req.InvoiceRef,
				InvoiceDate:        req.InvoiceDate,
				AmountHTCents:      req.AmountHTCents,
				VATCents:           req.VATCents,
			})
		},
	)

	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "get_trial_balance",
			Description: "Retrieve the general ledger trial balance (account balance lines and statement totals) for a legal entity and fiscal year.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"legal_entity_id"},
				"properties": map[string]any{
					"legal_entity_id":   map[string]any{"type": "string", "description": "Legal entity ID"},
					"entity_id":         map[string]any{"type": "string", "description": "Alias for legal_entity_id"},
					"fiscal_year":       map[string]any{"type": "integer", "description": "Fiscal year (e.g. 2025)"},
					"disabled_line_ids": map[string]any{"type": "array", "description": "Optional array of line_id or account_code strings to exclude"},
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
				return nil, fmt.Errorf("invalid get_trial_balance arguments: %w", err)
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
