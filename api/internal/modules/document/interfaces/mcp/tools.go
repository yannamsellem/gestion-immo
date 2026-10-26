package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yannamsellem/gestion-immo/internal/modules/document/application"
	"github.com/yannamsellem/gestion-immo/internal/platform/mcpserver"
)

// RegisterTools registers all atomic document and GLI dossier MCP tools on reg.
func RegisterTools(reg *mcpserver.Registry, svc *application.Service) {
	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "generate_document_from_template",
			Description: "Generate a document (.docx or PDF) from a registered template with entity, lease, tenant, and custom variables, and save it to Cloudflare R2.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"template_id"},
				"properties": map[string]any{
					"template_id":      map[string]any{"type": "string", "description": "Document template ID"},
					"legal_entity_id":  map[string]any{"type": "string", "description": "Optional legal entity ID"},
					"lease_id":         map[string]any{"type": "string", "description": "Optional lease ID"},
					"variables":        map[string]any{"type": "object", "description": "Key-value placeholders to interpolate"},
					"tables":           map[string]any{"type": "object", "description": "Dynamic table row slices keyed by table name"},
					"output_file_name": map[string]any{"type": "string", "description": "Optional output file name"},
				},
			},
		},
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var cmd application.GenerateDocCmd
			if err := json.Unmarshal(args, &cmd); err != nil {
				return nil, fmt.Errorf("invalid generate_document_from_template arguments: %w", err)
			}
			return svc.GenerateFromTemplate(ctx, cmd)
		},
	)

	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "generate_gli_dossier",
			Description: "Assemble a complete 1-click GLI (Garantie Loyers Impayés) claim ZIP archive for a lease in negative position (populated GLI declaration .docx, tenant ledger PDF, formal notice PDF, and R2 attachments).",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"lease_id"},
				"properties": map[string]any{
					"lease_id":    map[string]any{"type": "string", "description": "Lease ID in negative position"},
					"template_id": map[string]any{"type": "string", "description": "Optional GLI declaration template ID"},
				},
			},
		},
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var req struct {
				LeaseID    string `json:"lease_id"`
				TemplateID string `json:"template_id"`
			}
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, fmt.Errorf("invalid generate_gli_dossier arguments: %w", err)
			}
			return svc.AssembleGLIDossier(ctx, req.LeaseID, req.TemplateID)
		},
	)

	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "attach_file_to_r2",
			Description: "Upload and attach a document or photo (signed lease PDF, entry inspection, ID) to Cloudflare R2 and link it to a legal entity or lease.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"file_name"},
				"properties": map[string]any{
					"legal_entity_id": map[string]any{"type": "string", "description": "Optional legal entity ID"},
					"lease_id":        map[string]any{"type": "string", "description": "Optional lease ID"},
					"category":        map[string]any{"type": "string", "description": "File category (LEASE_SIGNED, EDL_ENTRANT, ID_LOCATAIRE, INVOICE, etc.)"},
					"file_name":       map[string]any{"type": "string", "description": "File name"},
					"content_type":    map[string]any{"type": "string", "description": "MIME content type"},
					"content_base64":  map[string]any{"type": "string", "description": "Base64-encoded file content"},
					"content_text":    map[string]any{"type": "string", "description": "Plain text content if not base64"},
				},
			},
		},
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var req struct {
				LegalEntityID string `json:"legal_entity_id"`
				LeaseID       string `json:"lease_id"`
				Category      string `json:"category"`
				FileName      string `json:"file_name"`
				ContentType   string `json:"content_type"`
				ContentBase64 string `json:"content_base64"`
				ContentText   string `json:"content_text"`
			}
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, fmt.Errorf("invalid attach_file_to_r2 arguments: %w", err)
			}
			var raw []byte
			if strings.TrimSpace(req.ContentBase64) != "" {
				decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(req.ContentBase64))
				if err != nil {
					return nil, fmt.Errorf("invalid content_base64: %w", err)
				}
				raw = decoded
			} else {
				raw = []byte(req.ContentText)
			}
			return svc.AttachFileToR2(ctx, application.AttachFileCmd{
				LegalEntityID: req.LegalEntityID,
				LeaseID:       req.LeaseID,
				Category:      req.Category,
				FileName:      req.FileName,
				ContentType:   req.ContentType,
				Content:       raw,
			})
		},
	)

	reg.RegisterTool(
		mcpserver.ToolDefinition{
			Name:        "search_documents",
			Description: "Search stored documents and attachments on R2 by legal entity ID, lease ID, or filename/category keyword.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"legal_entity_id": map[string]any{"type": "string", "description": "Legal entity ID"},
					"lease_id":        map[string]any{"type": "string", "description": "Lease ID"},
					"query":           map[string]any{"type": "string", "description": "Optional filename or category substring filter"},
				},
			},
		},
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var req struct {
				LegalEntityID string `json:"legal_entity_id"`
				LeaseID       string `json:"lease_id"`
				Query         string `json:"query"`
			}
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, fmt.Errorf("invalid search_documents arguments: %w", err)
			}
			return svc.SearchDocuments(ctx, req.LegalEntityID, req.LeaseID, req.Query)
		},
	)
}
