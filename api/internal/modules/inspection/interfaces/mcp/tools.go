package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yannamsellem/gestion-immo/internal/modules/inspection/application"
	"github.com/yannamsellem/gestion-immo/internal/platform/mcpserver"
)

// RegisterTools registers all atomic inspection and incident MCP tools on reg.
func RegisterTools(reg *mcpserver.Registry, svc *application.Service) {
	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "get_inspection_template",
			Description: "Get an inspection room/element template by ID, create one when name and rooms are provided, or list all registered inspection templates.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"template_id":      map[string]any{"type": "string", "description": "Optional inspection template ID to retrieve"},
					"name":             map[string]any{"type": "string", "description": "Optional template name when creating a new template"},
					"rooms":            map[string]any{"type": "array", "description": "Optional room list when creating a new template"},
					"default_elements": map[string]any{"type": "array", "description": "Optional default elements per room"},
				},
			},
		},
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var req struct {
				TemplateID      string   `json:"template_id"`
				Name            string   `json:"name"`
				Rooms           []string `json:"rooms"`
				DefaultElements []string `json:"default_elements"`
			}
			if len(args) > 0 {
				if err := json.Unmarshal(args, &req); err != nil {
					return nil, fmt.Errorf("invalid get_inspection_template arguments: %w", err)
				}
			}
			if strings.TrimSpace(req.TemplateID) != "" {
				return svc.GetTemplate(ctx, req.TemplateID)
			}
			if strings.TrimSpace(req.Name) != "" && len(req.Rooms) > 0 {
				return svc.CreateTemplate(ctx, application.CreateInspectionTemplateCmd{
					Name:            req.Name,
					Rooms:           req.Rooms,
					DefaultElements: req.DefaultElements,
				})
			}
			return svc.ListTemplates(ctx)
		},
	)

	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "create_inspection",
			Description: "Create a Move-In (MOVE_IN) or Move-Out (MOVE_OUT) property inspection for a lease. EntryInspectionID is optional for standalone exit inspections.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"lease_id", "template_id", "type"},
				"properties": map[string]any{
					"lease_id":            map[string]any{"type": "string", "description": "Lease ID"},
					"template_id":         map[string]any{"type": "string", "description": "Inspection template ID"},
					"type":                map[string]any{"type": "string", "description": "MOVE_IN or MOVE_OUT"},
					"entry_inspection_id": map[string]any{"type": "string", "description": "Optional structured entry inspection ID for exit comparison"},
					"inspection_date":     map[string]any{"type": "string", "description": "Optional inspection date (YYYY-MM-DD)"},
				},
			},
		},
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var cmd application.CreateInspectionCmd
			if err := json.Unmarshal(args, &cmd); err != nil {
				return nil, fmt.Errorf("invalid create_inspection arguments: %w", err)
			}
			return svc.CreateInspection(ctx, cmd)
		},
	)

	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "upsert_inspection_items",
			Description: "Populate or update room-by-room element conditions (NEW, GOOD, FAIR_WEAR, POOR, OUT_OF_SERVICE), observations, and R2 photo keys for an inspection, and optionally generate the PDF report.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"inspection_id", "items"},
				"properties": map[string]any{
					"inspection_id":   map[string]any{"type": "string", "description": "Inspection ID"},
					"items":           map[string]any{"type": "array", "description": "Array of {room_name, element_name, condition, observation, photo_r2_keys}"},
					"generate_report": map[string]any{"type": "boolean", "description": "If true, also generates the PDF inspection report and entry/exit comparison"},
				},
			},
		},
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var req struct {
				InspectionID   string                            `json:"inspection_id"`
				Items          []application.InspectionItemInput `json:"items"`
				GenerateReport bool                              `json:"generate_report"`
			}
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, fmt.Errorf("invalid upsert_inspection_items arguments: %w", err)
			}
			updated, err := svc.UpsertInspectionItems(ctx, req.InspectionID, req.Items)
			if err != nil {
				return nil, err
			}
			if req.GenerateReport {
				return svc.GenerateInspectionReport(ctx, req.InspectionID)
			}
			return updated, nil
		},
	)

	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "create_incident",
			Description: "Open a new property, lot, or lease incident / insurance claim (OPEN, INSURANCE_CLAIM, WORK_IN_PROGRESS, CLOSED) with observations, estimated cost, and R2 photo keys.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"title"},
				"properties": map[string]any{
					"property_id":            map[string]any{"type": "string", "description": "Property ID"},
					"lot_id":                 map[string]any{"type": "string", "description": "Lot ID"},
					"lease_id":               map[string]any{"type": "string", "description": "Optional Lease ID"},
					"title":                  map[string]any{"type": "string", "description": "Incident title / description"},
					"status":                 map[string]any{"type": "string", "description": "OPEN, INSURANCE_CLAIM, WORK_IN_PROGRESS, or CLOSED"},
					"occurred_at":            map[string]any{"type": "string", "description": "Date of occurrence (YYYY-MM-DD)"},
					"insurance_claim_number": map[string]any{"type": "string", "description": "Optional insurance claim reference"},
					"estimated_cost_cents":   map[string]any{"type": "integer", "description": "Estimated repair cost in euro cents"},
					"observations":           map[string]any{"type": "array", "description": "Initial observations"},
					"photo_r2_keys":          map[string]any{"type": "array", "description": "R2 object keys of incident photos"},
				},
			},
		},
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var cmd application.CreateIncidentCmd
			if err := json.Unmarshal(args, &cmd); err != nil {
				return nil, fmt.Errorf("invalid create_incident arguments: %w", err)
			}
			return svc.CreateIncident(ctx, cmd)
		},
	)

	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "update_incident",
			Description: "Update an existing incident's status (OPEN, INSURANCE_CLAIM, WORK_IN_PROGRESS, CLOSED), insurance claim number, estimated cost, observations, or R2 photo keys.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"incident_id"},
				"properties": map[string]any{
					"incident_id":            map[string]any{"type": "string", "description": "Incident ID"},
					"status":                 map[string]any{"type": "string", "description": "OPEN, INSURANCE_CLAIM, WORK_IN_PROGRESS, or CLOSED"},
					"insurance_claim_number": map[string]any{"type": "string", "description": "Insurance claim reference"},
					"estimated_cost_cents":   map[string]any{"type": "integer", "description": "Updated estimated cost in euro cents"},
					"add_observation":        map[string]any{"type": "string", "description": "Observation note to append"},
					"observations":           map[string]any{"type": "array", "description": "Full replacement list of observations"},
					"add_photo_r2_keys":      map[string]any{"type": "array", "description": "Additional R2 photo keys to append"},
					"photo_r2_keys":          map[string]any{"type": "array", "description": "Full replacement list of R2 photo keys"},
				},
			},
		},
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var cmd application.UpdateIncidentCmd
			if err := json.Unmarshal(args, &cmd); err != nil {
				return nil, fmt.Errorf("invalid update_incident arguments: %w", err)
			}
			return svc.UpdateIncident(ctx, cmd)
		},
	)
}
