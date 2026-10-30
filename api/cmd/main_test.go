package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/yannamsellem/gestion-immo/cmd/bootstrap"
	"github.com/yannamsellem/gestion-immo/cmd/server"
	"github.com/yannamsellem/gestion-immo/internal/platform/config"
)

func TestBootstrapSuperAdmin_ValidatesRequiredFlagsBeforeDatabaseConnection(t *testing.T) {
	var outBuf bytes.Buffer
	app := &cli.Command{
		Name:   "gestion-immo",
		Writer: &outBuf,
		Commands: []*cli.Command{
			bootstrap.Command(),
		},
	}

	if err := app.Run(context.Background(), []string{"gestion-immo", "bootstrap", "super-admin", "--email", "yann@example.com"}); err == nil {
		t.Fatal("expected bootstrap super-admin without required flags to fail")
	}
}

func TestServer_ExportOpenAPIFlag_WritesValidOpenAPI31SpecWithoutDatabase(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "openapi.json")
	app := &cli.Command{
		Name: "gestion-immo",
		Commands: []*cli.Command{
			server.Command(),
		},
	}

	if err := app.Run(context.Background(), []string{"gestion-immo", "server", "--export-openapi", outPath}); err != nil {
		t.Fatalf("server --export-openapi failed: %v", err)
	}

	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read exported openapi.json: %v", err)
	}

	var spec struct {
		OpenAPI    string                    `json:"openapi"`
		Paths      map[string]map[string]any `json:"paths"`
		Components struct {
			Schemas map[string]any `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("unmarshal exported openapi.json: %v", err)
	}
	if !strings.HasPrefix(spec.OpenAPI, "3.1") {
		t.Fatalf("expected OpenAPI 3.1.x, got %q", spec.OpenAPI)
	}

	requiredPaths := []string{
		"/api/v1/auth/login",
		"/api/v1/entities",
		"/api/v1/leases",
		"/api/v1/payments/cash",
		"/api/v1/documents/generate",
		"/api/v1/inspections",
		"/api/v1/entities/{id}/fec",
	}
	for _, p := range requiredPaths {
		if _, ok := spec.Paths[p]; !ok {
			t.Errorf("missing expected OpenAPI path %q", p)
		}
	}
}

func TestServerHTTPHandler_EnforcesJWTAuthOnBusinessRoutes(t *testing.T) {
	handler, err := server.BuildHTTPHandler(nil, config.Config{})
	if err != nil {
		t.Fatalf("BuildHTTPHandler failed: %v", err)
	}

	reqHealth := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recHealth := httptest.NewRecorder()
	handler.ServeHTTP(recHealth, reqHealth)
	if recHealth.Code != http.StatusOK {
		t.Fatalf("expected GET /healthz to return 200 OK, got %d", recHealth.Code)
	}

	reqEntities := httptest.NewRequest(http.MethodGet, "/api/v1/entities", nil)
	recEntities := httptest.NewRecorder()
	handler.ServeHTTP(recEntities, reqEntities)
	if recEntities.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated GET /api/v1/entities to return 401 Unauthorized, got %d: %s", recEntities.Code, recEntities.Body.String())
	}
}
