package mcpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/yannamsellem/gestion-immo/internal/modules/accounting"
	"github.com/yannamsellem/gestion-immo/internal/modules/document"
	"github.com/yannamsellem/gestion-immo/internal/modules/inspection"
	"github.com/yannamsellem/gestion-immo/internal/modules/lease"
	"github.com/yannamsellem/gestion-immo/internal/modules/payment"
	"github.com/yannamsellem/gestion-immo/internal/modules/portfolio"
	"github.com/yannamsellem/gestion-immo/internal/platform/mcpserver"
	"github.com/yannamsellem/gestion-immo/internal/platform/r2"
	"github.com/yannamsellem/gestion-immo/internal/platform/templating"
)

func init() {
	mcpserver.RegisterTestFullAppToolsHook(func(reg *mcpserver.Registry) {
		r2Store := r2.NewInMemoryStorage()
		pdfEngine := templating.NewInMemoryPdfEngine()

		portfolioMod := portfolio.NewModule(nil)
		leaseMod := lease.NewModule(nil)
		paymentMod := payment.NewModule(nil, leaseMod.Service(), false)
		documentMod := document.NewModule(
			nil,
			r2Store,
			pdfEngine,
			portfolioMod.Service(),
			leaseMod.Service(),
			paymentMod.Service(),
		)
		inspectionMod := inspection.NewModule(nil, r2Store, pdfEngine)
		accountingMod := accounting.NewModule(nil, r2Store, pdfEngine)

		portfolioMod.RegisterMCPTools(reg)
		leaseMod.RegisterMCPTools(reg)
		paymentMod.RegisterMCPTools(reg)
		documentMod.RegisterMCPTools(reg)
		inspectionMod.RegisterMCPTools(reg)
		accountingMod.RegisterMCPTools(reg)
	})
}

func TestMCPRegistry_RequiresBearerAndExposesAllModuleToolsWithoutMigrateDriveRecord(t *testing.T) {
	srv := mcpserver.NewTestFullAppRegistry(t, "mcp_valid_token_123")

	reqUnauth := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	recUnauth := httptest.NewRecorder()
	srv.ServeHTTP(recUnauth, reqUnauth)
	if recUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without bearer token, got %d", recUnauth.Code)
	}

	reqList := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	reqList.Header.Set("Authorization", "Bearer mcp_valid_token_123")
	recList := httptest.NewRecorder()
	srv.ServeHTTP(recList, reqList)
	if recList.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recList.Code)
	}
	body := recList.Body.String()
	for _, requiredTool := range []string{"create_legal_entity", "record_cash_payment", "list_negative_positions", "upsert_inspection_items", "generate_gli_dossier", "preview_financial_statement"} {
		if !strings.Contains(body, requiredTool) {
			t.Fatalf("missing required MCP tool %q in %s", requiredTool, body)
		}
	}
	if strings.Contains(body, "migrate_drive_record") {
		t.Fatal("migrate_drive_record must not exist")
	}
}

func TestMCPRegistry_StreamableHTTPInitializeAndToolExecution(t *testing.T) {
	srv := mcpserver.NewTestFullAppRegistry(t, "mcp_valid_token_123")

	// 1. GET /mcp health/metadata endpoint with valid Bearer token
	reqGet := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	reqGet.Header.Set("Authorization", "Bearer mcp_valid_token_123")
	recGet := httptest.NewRecorder()
	srv.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("expected 200 on GET /mcp, got %d", recGet.Code)
	}

	// 2. POST /mcp initialize
	reqInit := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"antigravity","version":"1.0.0"}}}`))
	reqInit.Header.Set("Authorization", "Bearer mcp_valid_token_123")
	recInit := httptest.NewRecorder()
	srv.ServeHTTP(recInit, reqInit)
	if recInit.Code != http.StatusOK || !strings.Contains(recInit.Body.String(), "gestion-immo-mcp") {
		t.Fatalf("unexpected initialize response: %d %s", recInit.Code, recInit.Body.String())
	}

	// 2b. Verify OAuth Protected Resource Metadata handler (oauthex + auth)
	reqMeta := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", nil)
	recMeta := httptest.NewRecorder()
	srv.ProtectedResourceMetadataHandler().ServeHTTP(recMeta, reqMeta)
	if recMeta.Code != http.StatusOK || !strings.Contains(recMeta.Body.String(), "gestion-immo-mcp") {
		t.Fatalf("unexpected protected resource metadata response: %d %s", recMeta.Code, recMeta.Body.String())
	}

	// 3. POST /mcp tools/call -> create_legal_entity
	callPayload := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"create_legal_entity","arguments":{"name":"SCI HORIZON","entity_type":"SCI_IS","share_capital_cents":100000,"total_shares":100}}}`
	reqCall := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(callPayload))
	reqCall.Header.Set("Authorization", "Bearer mcp_valid_token_123")
	recCall := httptest.NewRecorder()
	srv.ServeHTTP(recCall, reqCall)
	if recCall.Code != http.StatusOK {
		t.Fatalf("expected 200 on tools/call, got %d: %s", recCall.Code, recCall.Body.String())
	}
	var rpcResp struct {
		Result struct {
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	if err := json.Unmarshal(recCall.Body.Bytes(), &rpcResp); err != nil {
		t.Fatalf("unmarshal tools/call response: %v", err)
	}
	if rpcResp.Result.IsError {
		t.Fatalf("expected non-error tool call result, got %s", recCall.Body.String())
	}

	// 4. Verify native mcp.Client <-> mcp.Server session over mcp.NewInMemoryTransports()
	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := srv.Server().Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("mcp.Server.Connect failed: %v", err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "antigravity-test-client", Version: "1.0.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("mcp.Client.Connect failed: %v", err)
	}
	defer clientSession.Close()

	listRes, err := clientSession.ListTools(ctx, nil)
	if err != nil || len(listRes.Tools) == 0 {
		t.Fatalf("clientSession.ListTools failed (len=%d): %v", len(listRes.Tools), err)
	}
}
