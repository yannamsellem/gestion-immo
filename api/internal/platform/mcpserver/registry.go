package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/oauthex"

	"github.com/yannamsellem/gestion-immo/internal/platform/httpx"
)

// ToolDefinition describes an atomic MCP tool exposed to Antigravity.
type ToolDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

// ToolHandlerFunc executes an MCP tool call with raw JSON arguments.
type ToolHandlerFunc func(ctx context.Context, args json.RawMessage) (any, error)

// TokenValidatorFunc validates a raw Bearer token and returns the authenticated userID or an error.
type TokenValidatorFunc func(ctx context.Context, rawBearer string) (string, error)

type contextKey string

const userIDContextKey contextKey = "mcp_user_id"

// UserIDFromContext returns the authenticated user ID stored in ctx by the MCP server, if any.
func UserIDFromContext(ctx context.Context) string {
	if ti := auth.TokenInfoFromContext(ctx); ti != nil && ti.UserID != "" {
		return ti.UserID
	}
	val, _ := ctx.Value(userIDContextKey).(string)
	return val
}

type registeredTool struct {
	definition ToolDefinition
	handler    ToolHandlerFunc
}

// Registry wraps the official github.com/modelcontextprotocol/go-sdk/mcp Server,
// StreamableHTTPHandler, and auth/oauthex Bearer middleware.
type Registry struct {
	mu                sync.RWMutex
	validator         TokenValidatorFunc
	order             []string
	tools             map[string]registeredTool
	mcpServer         *mcp.Server
	streamableHandler *mcp.StreamableHTTPHandler
	authedHandler     http.Handler
	metadataHandler   http.Handler
}

// NewRegistry constructs a new MCP Registry backed by modelcontextprotocol/go-sdk
// and protected by auth.RequireBearerToken.
func NewRegistry(validator TokenValidatorFunc) *Registry {
	mcpSrv := mcp.NewServer(&mcp.Implementation{
		Name:    "gestion-immo-mcp",
		Version: "1.0.0",
	}, nil)

	streamable := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return mcpSrv },
		&mcp.StreamableHTTPOptions{
			Stateless:                  true,
			JSONResponse:               true,
			DisableLocalhostProtection: true,
		},
	)

	resourceMetadata := &oauthex.ProtectedResourceMetadata{
		Resource:               "gestion-immo-mcp",
		ResourceName:           "Gestion Immo MCP Server",
		ScopesSupported:        []string{"mcp:tools"},
		BearerMethodsSupported: []string{"header"},
	}

	reg := &Registry{
		validator:         validator,
		tools:             make(map[string]registeredTool),
		mcpServer:         mcpSrv,
		streamableHandler: streamable,
		metadataHandler:   auth.ProtectedResourceMetadataHandler(resourceMetadata),
	}

	verifier := func(ctx context.Context, token string, _ *http.Request) (*auth.TokenInfo, error) {
		if reg.validator == nil {
			return nil, fmt.Errorf("%w: missing token validator", auth.ErrInvalidToken)
		}
		userID, err := reg.validator(ctx, strings.TrimSpace(token))
		if err != nil {
			return nil, fmt.Errorf("%w: %v", auth.ErrInvalidToken, err)
		}
		userID = strings.TrimSpace(userID)
		if userID == "" {
			return nil, fmt.Errorf("%w: empty user id", auth.ErrInvalidToken)
		}
		return &auth.TokenInfo{
			UserID: userID,
			Scopes: []string{"mcp:tools"},
		}, nil
	}

	bearerMiddleware := auth.RequireBearerToken(verifier, &auth.RequireBearerTokenOptions{
		ResourceMetadataURL:    "/.well-known/oauth-protected-resource",
		Scopes:                 []string{"mcp:tools"},
		AllowMissingExpiration: true,
	})

	reg.authedHandler = bearerMiddleware(http.HandlerFunc(reg.serveAuthenticated))
	return reg
}

// Server returns the underlying *mcp.Server instance.
func (reg *Registry) Server() *mcp.Server {
	return reg.mcpServer
}

// ProtectedResourceMetadataHandler returns the RFC 9728 OAuth 2.0 protected resource metadata handler.
func (reg *Registry) ProtectedResourceMetadataHandler() http.Handler {
	return reg.metadataHandler
}

// RegisterTool registers or replaces an MCP tool definition and its handler on the underlying *mcp.Server.
func (reg *Registry) RegisterTool(def ToolDefinition, handler ToolHandlerFunc) {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	if def.InputSchema == nil {
		def.InputSchema = map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}
	if _, exists := reg.tools[def.Name]; !exists {
		reg.order = append(reg.order, def.Name)
	}
	reg.tools[def.Name] = registeredTool{
		definition: def,
		handler:    handler,
	}

	reg.mcpServer.AddTool(
		&mcp.Tool{
			Name:        def.Name,
			Description: def.Description,
			InputSchema: def.InputSchema,
		},
		func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if ti := auth.TokenInfoFromContext(ctx); ti != nil && ti.UserID != "" {
				ctx = context.WithValue(ctx, userIDContextKey, ti.UserID)
			}

			var rawArgs json.RawMessage
			if req != nil && req.Params != nil && len(req.Params.Arguments) > 0 {
				rawArgs = req.Params.Arguments
			} else {
				rawArgs = json.RawMessage(`{}`)
			}

			res, callErr := handler(ctx, rawArgs)
			if callErr != nil {
				errResult := &mcp.CallToolResult{}
				errResult.SetError(callErr)
				return errResult, nil
			}

			serialized, err := json.Marshal(res)
			if err != nil {
				errResult := &mcp.CallToolResult{}
				errResult.SetError(err)
				return errResult, nil
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: string(serialized)},
				},
				StructuredContent: toStructuredJSONObject(serialized),
			}, nil
		},
	)
}

func toStructuredJSONObject(serialized []byte) any {
	var obj map[string]any
	if err := json.Unmarshal(serialized, &obj); err == nil && obj != nil {
		return obj
	}
	var arr []any
	if err := json.Unmarshal(serialized, &arr); err == nil && arr != nil {
		return map[string]any{"items": arr}
	}
	return nil
}

// Tools returns a snapshot of all registered tool definitions in registration order.
func (reg *Registry) Tools() []ToolDefinition {
	reg.mu.RLock()
	defer reg.mu.RUnlock()

	out := make([]ToolDefinition, 0, len(reg.order))
	for _, name := range reg.order {
		out = append(out, reg.tools[name].definition)
	}
	return out
}

// ServeHTTP implements http.Handler for the Streamable HTTP MCP endpoint (/mcp),
// enforcing Bearer authentication via github.com/modelcontextprotocol/go-sdk/auth.
func (reg *Registry) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reg.authedHandler.ServeHTTP(w, r)
}

func (reg *Registry) serveAuthenticated(w http.ResponseWriter, r *http.Request) {
	if ti := auth.TokenInfoFromContext(r.Context()); ti != nil && ti.UserID != "" {
		r = r.WithContext(context.WithValue(r.Context(), userIDContextKey, ti.UserID))
	}

	if r.Method == http.MethodGet {
		httpx.OK(w, map[string]any{
			"server":   "gestion-immo-mcp",
			"protocol": "2025-03-26",
			"status":   "ready",
		})
		return
	}

	if r.Method == http.MethodPost {
		if strings.TrimSpace(r.Header.Get("Content-Type")) == "" {
			r.Header.Set("Content-Type", "application/json")
		}
		accept := r.Header.Get("Accept")
		if !strings.Contains(accept, "application/json") || !strings.Contains(accept, "text/event-stream") {
			r.Header.Set("Accept", "application/json, text/event-stream")
		}
	}

	reg.streamableHandler.ServeHTTP(w, r)
}
