package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
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
		},
		Action: run,
	}
}

func run(ctx context.Context, cmd *cli.Command) error {
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
	accountingMod := accounting.NewModule(pool, r2Storage, pdfEngine)

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
	portfolioMod.RegisterHTTPRoutes(protectedMux)
	leaseMod.RegisterHTTPRoutes(protectedMux)
	paymentMod.RegisterHTTPRoutes(protectedMux)
	documentMod.RegisterHTTPRoutes(protectedMux)
	inspectionMod.RegisterHTTPRoutes(protectedMux)
	accountingMod.RegisterHTTPRoutes(protectedMux)
	mux.Handle("/api/v1/", authMod.RequireAuthMiddleware()(protectedMux))

	mux.Handle("POST /mcp", mcpRegistry)
	mux.Handle("GET /mcp", mcpRegistry)
	mux.Handle("GET /.well-known/oauth-protected-resource", mcpRegistry.ProtectedResourceMetadataHandler())

	return mux, nil
}

func serveHTTP(ctx context.Context, addr string, handler http.Handler) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
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
