package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yannamsellem/gestion-immo/internal/modules/payment/application"
	"github.com/yannamsellem/gestion-immo/internal/platform/mcpserver"
)

// RegisterTools registers all atomic payment MCP tools on reg.
func RegisterTools(reg *mcpserver.Registry, svc *application.Service) {
	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "record_cash_payment",
			Description: "Record a tenant cash rent payment, allocate it across unpaid rent calls (explicit month first, FIFO fallback), and post a debit entry in the PCG 530 Cash Journal.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"lease_id", "amount_cents"},
				"properties": map[string]any{
					"lease_id":       map[string]any{"type": "string", "description": "Lease ID"},
					"amount_cents":   map[string]any{"type": "integer", "description": "Cash payment amount in euro cents"},
					"label":          map[string]any{"type": "string", "description": "Payment label (may include an explicit French month name or MM/YYYY)"},
					"receipt_number": map[string]any{"type": "string", "description": "Optional cash receipt number"},
					"payment_date":   map[string]any{"type": "string", "description": "Payment date (YYYY-MM-DD)"},
				},
			},
		},
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var cmd application.RecordCashPaymentCmd
			if err := json.Unmarshal(args, &cmd); err != nil {
				return nil, fmt.Errorf("invalid record_cash_payment arguments: %w", err)
			}
			return svc.RecordCashPayment(ctx, cmd)
		},
	)

	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "record_bank_payment",
			Description: "Record a bank transfer, check, or direct debit rent payment and allocate it across unpaid rent calls (explicit month in transfer label first, FIFO fallback).",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"lease_id", "amount_cents"},
				"properties": map[string]any{
					"lease_id":     map[string]any{"type": "string", "description": "Lease ID"},
					"amount_cents": map[string]any{"type": "integer", "description": "Payment amount in euro cents"},
					"method":       map[string]any{"type": "string", "enum": []string{"BANK_TRANSFER", "CHECK", "DIRECT_DEBIT"}},
					"label":        map[string]any{"type": "string", "description": "Bank transfer label (e.g., 'Virement loyer mars 2026')"},
					"external_ref": map[string]any{"type": "string", "description": "Optional bank reference"},
					"payment_date": map[string]any{"type": "string", "description": "Payment date (YYYY-MM-DD)"},
				},
			},
		},
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var cmd application.RecordBankPaymentCmd
			if err := json.Unmarshal(args, &cmd); err != nil {
				return nil, fmt.Errorf("invalid record_bank_payment arguments: %w", err)
			}
			return svc.RecordBankPayment(ctx, cmd)
		},
	)

	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "list_negative_positions",
			Description: "List all leases in negative position (impayés de loyers) for a legal entity, including unpaid months and GLI policy eligibility.",
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
				return nil, fmt.Errorf("invalid list_negative_positions arguments: %w", err)
			}
			entityID := strings.TrimSpace(req.LegalEntityID)
			if entityID == "" {
				entityID = strings.TrimSpace(req.EntityID)
			}
			return svc.ListNegativePositions(ctx, entityID)
		},
	)
}
