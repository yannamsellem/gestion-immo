package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
)

var (
	testFullAppToolsHookMu sync.RWMutex
	testFullAppToolsHook   func(reg *Registry)
)

// RegisterTestFullAppToolsHook registers a callback used by NewTestFullAppRegistry to wire
// all domain module MCP tools without introducing an import cycle into package mcpserver.
func RegisterTestFullAppToolsHook(fn func(reg *Registry)) {
	testFullAppToolsHookMu.Lock()
	defer testFullAppToolsHookMu.Unlock()
	testFullAppToolsHook = fn
}

// NewTestFullAppRegistry constructs a test MCP Registry validating validBearerToken and exposing
// all domain module MCP tools (strictly excluding migrate_drive_record).
func NewTestFullAppRegistry(t testing.TB, validBearerToken string) *Registry {
	t.Helper()
	expected := strings.TrimSpace(validBearerToken)
	reg := NewRegistry(func(_ context.Context, rawBearer string) (string, error) {
		if expected != "" && strings.TrimSpace(rawBearer) == expected {
			return "usr_test_mcp", nil
		}
		return "", errors.New("invalid MCP bearer token")
	})

	testFullAppToolsHookMu.RLock()
	hook := testFullAppToolsHook
	testFullAppToolsHookMu.RUnlock()

	if hook != nil {
		hook(reg)
		return reg
	}

	fallbackTools := []string{
		"create_legal_entity",
		"upsert_shareholder",
		"list_entities",
		"get_entity_summary",
		"create_property",
		"create_lot",
		"list_properties",
		"create_tenant",
		"create_lease",
		"list_leases",
		"record_cash_payment",
		"record_bank_payment",
		"list_negative_positions",
		"generate_document_from_template",
		"generate_gli_dossier",
		"attach_file_to_r2",
		"search_documents",
		"get_inspection_template",
		"create_inspection",
		"upsert_inspection_items",
		"create_incident",
		"update_incident",
		"preview_financial_statement",
		"generate_annual_accounts_pdf",
		"export_fec",
	}
	for _, name := range fallbackTools {
		toolName := name
		reg.RegisterTool(
			ToolDefinition{
				Name:        toolName,
				Description: "Atomic MCP tool " + toolName,
			},
			func(_ context.Context, _ json.RawMessage) (any, error) {
				return map[string]any{"tool": toolName, "ok": true}, nil
			},
		)
	}
	return reg
}
