package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/urfave/cli/v3"

	"github.com/yannamsellem/gestion-immo/internal/modules/accounting"
	"github.com/yannamsellem/gestion-immo/internal/modules/auth"
	"github.com/yannamsellem/gestion-immo/internal/modules/document"
	"github.com/yannamsellem/gestion-immo/internal/modules/inspection"
	"github.com/yannamsellem/gestion-immo/internal/modules/lease"
	"github.com/yannamsellem/gestion-immo/internal/modules/payment"
	"github.com/yannamsellem/gestion-immo/internal/modules/portfolio"
	"github.com/yannamsellem/gestion-immo/internal/platform/config"
	"github.com/yannamsellem/gestion-immo/internal/platform/httpx"
	"github.com/yannamsellem/gestion-immo/internal/platform/mcpserver"
	"github.com/yannamsellem/gestion-immo/internal/platform/postgres"
	"github.com/yannamsellem/gestion-immo/internal/platform/r2"
	"github.com/yannamsellem/gestion-immo/internal/platform/templating"
)

// Command returns the CLI command for running the HTTP REST API and Streamable HTTP MCP server.
func Command() *cli.Command {
	return &cli.Command{
		Name:    "server",
		Aliases: []string{"serve"},
		Usage:   "Run the HTTP REST API and Streamable HTTP MCP server (/mcp)",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "port",
				Usage:   "HTTP listen port",
				Value:   "8080",
				Sources: cli.EnvVars("PORT"),
			},
			&cli.StringFlag{
				Name:  "export-openapi",
				Usage: "Write the OpenAPI 3.1 JSON specification to the given file path and exit without starting the server",
			},
		},
		Action: run,
	}
}

func run(ctx context.Context, cmd *cli.Command) error {
	if exportPath := strings.TrimSpace(cmd.String("export-openapi")); exportPath != "" {
		return exportOpenAPISpec(exportPath)
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	handler, err := BuildHTTPHandler(pool, cfg)
	if err != nil {
		return err
	}

	port := strings.TrimSpace(cmd.String("port"))
	return serveHTTP(ctx, ":"+port, handler)
}

func exportOpenAPISpec(outPath string) error {
	r2Storage := r2.NewInMemoryStorage()
	pdfEngine := templating.NewInMemoryPdfEngine()

	authMod, err := auth.NewModule(nil, "", "")
	if err != nil {
		return err
	}
	portfolioMod := portfolio.NewModule(nil)
	leaseMod := lease.NewModule(nil)
	paymentMod := payment.NewModule(nil, leaseMod.Service(), false)
	documentMod := document.NewModule(
		nil,
		r2Storage,
		pdfEngine,
		portfolioMod.Service(),
		leaseMod.Service(),
		paymentMod.Service(),
	)
	inspectionMod := inspection.NewModule(nil, r2Storage, pdfEngine)
	accountingMod := accounting.NewModule(nil, r2Storage, pdfEngine, portfolioMod.Service())

	mux := http.NewServeMux()
	api := httpx.NewHumaAPI(mux)
	authMod.RegisterHumaRoutes(api)
	portfolioMod.RegisterHumaRoutes(api)
	leaseMod.RegisterHumaRoutes(api)
	paymentMod.RegisterHumaRoutes(api)
	documentMod.RegisterHumaRoutes(api)
	inspectionMod.RegisterHumaRoutes(api)
	accountingMod.RegisterHumaRoutes(api)

	raw, err := json.MarshalIndent(api.OpenAPI(), "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')

	if dir := filepath.Dir(outPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(outPath, raw, 0o644)
}

// BuildHTTPHandler constructs the root HTTP and MCP handler with JWT authentication enforced on all business routes.
func BuildHTTPHandler(pool *pgxpool.Pool, cfg config.Config) (http.Handler, error) {
	r2Storage := r2.NewS3Storage(
		cfg.R2Endpoint,
		cfg.R2AccessKeyID,
		cfg.R2SecretAccessKey.Value(),
		cfg.R2BucketName,
	)
	pdfEngine := templating.NewPdfEngine(cfg.GotenbergURL)

	authMod, err := auth.NewModule(pool, cfg.JWTKeysDir, cfg.JWTKID)
	if err != nil {
		return nil, err
	}
	portfolioMod := portfolio.NewModule(pool)
	leaseMod := lease.NewModule(pool)
	paymentMod := payment.NewModule(pool, leaseMod.Service(), cfg.EnableOpenBanking)
	documentMod := document.NewModule(
		pool,
		r2Storage,
		pdfEngine,
		portfolioMod.Service(),
		leaseMod.Service(),
		paymentMod.Service(),
	)
	inspectionMod := inspection.NewModule(pool, r2Storage, pdfEngine)
	accountingMod := accounting.NewModule(pool, r2Storage, pdfEngine, portfolioMod.Service())
	leaseMod.Service().WithAccountingRecorder(accountingMod.Service())
	paymentMod.Service().WithAccountingRecorder(accountingMod.Service())

	mcpRegistry := mcpserver.NewRegistry(authMod.Service().ValidateMCPBearer)
	portfolioMod.RegisterMCPTools(mcpRegistry)
	leaseMod.RegisterMCPTools(mcpRegistry)
	paymentMod.RegisterMCPTools(mcpRegistry)
	documentMod.RegisterMCPTools(mcpRegistry)
	inspectionMod.RegisterMCPTools(mcpRegistry)
	accountingMod.RegisterMCPTools(mcpRegistry)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		httpx.OK(w, map[string]string{"status": "ok"})
	})
	authMod.RegisterHTTPRoutes(mux)

	protectedMux := http.NewServeMux()
	protectedAPI := httpx.NewHumaAPI(protectedMux)
	portfolioMod.RegisterHumaRoutes(protectedAPI)
	leaseMod.RegisterHumaRoutes(protectedAPI)
	paymentMod.RegisterHumaRoutes(protectedAPI)
	documentMod.RegisterHumaRoutes(protectedAPI)
	inspectionMod.RegisterHumaRoutes(protectedAPI)
	accountingMod.RegisterHumaRoutes(protectedAPI)
	mux.Handle("/api/v1/", authMod.RequireAuthMiddleware()(protectedMux))

	mux.Handle("POST /mcp", mcpRegistry)
	mux.Handle("GET /mcp", mcpRegistry)
	mux.Handle("GET /.well-known/oauth-protected-resource", mcpRegistry.ProtectedResourceMetadataHandler())

	return httpx.RequestLogger(mux), nil
}

func serveHTTP(ctx context.Context, addr string, handler http.Handler) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("starting gestion-immo HTTP & MCP server", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("shutting down gestion-immo HTTP & MCP server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
