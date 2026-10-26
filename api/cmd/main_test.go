package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/yannamsellem/gestion-immo/cmd/bootstrap"
	"github.com/yannamsellem/gestion-immo/cmd/migrate"
	"github.com/yannamsellem/gestion-immo/cmd/server"
	"github.com/yannamsellem/gestion-immo/cmd/token"
	"github.com/yannamsellem/gestion-immo/internal/modules/auth/domain"
)

func TestCLIApp_ExposesRequiredSubcommandsAndFlags(t *testing.T) {
	var outBuf bytes.Buffer
	app := &cli.Command{
		Name:   "gestion-immo",
		Usage:  "Gestion Immobilière & Comptable — Go Modular Monolith + Streamable HTTP MCP Server",
		Writer: &outBuf,
		Commands: []*cli.Command{
			server.Command(),
			bootstrap.Command(),
			migrate.Command(),
			token.Command(),
		},
	}

	for _, expectedCmd := range []string{"server", "bootstrap", "migrate", "token"} {
		if app.Command(expectedCmd) == nil {
			t.Fatalf("missing top-level CLI command %q", expectedCmd)
		}
	}

	migrateCmd := app.Command("migrate")
	if migrateCmd.Command("up") == nil || migrateCmd.Command("down") == nil {
		t.Fatal("migrate command must expose 'up' and 'down' subcommands")
	}

	bootstrapCmd := app.Command("bootstrap")
	if bootstrapCmd.Command("super-admin") == nil || bootstrapCmd.Command("approve") == nil || bootstrapCmd.Command("reject") == nil || bootstrapCmd.Command("pending") == nil {
		t.Fatal("bootstrap command must expose 'super-admin', 'approve', 'reject', and 'pending' subcommands")
	}

	tokenCmd := app.Command("token")
	if tokenCmd.Command("create") == nil || tokenCmd.Command("keygen") == nil {
		t.Fatal("token command must expose 'create' and 'keygen' subcommands")
	}

	// Verify bootstrap super-admin validates required flags before opening any DB connection
	if err := app.Run(context.Background(), []string{"gestion-immo", "bootstrap", "super-admin", "--email", "yann@example.com"}); err == nil {
		t.Fatal("expected bootstrap super-admin without required flags to fail")
	}
}

func TestTokenKeygenCommand(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "keys")
	var outBuf bytes.Buffer
	app := &cli.Command{
		Name:   "gestion-immo",
		Writer: &outBuf,
		Commands: []*cli.Command{
			token.Command(),
		},
	}

	err := app.Run(context.Background(), []string{"gestion-immo", "token", "keygen", "--out-dir", outDir, "--kid", "test-kid"})
	if err != nil {
		t.Fatalf("token keygen failed: %v", err)
	}

	privInfo, err := os.Stat(filepath.Join(outDir, "private.pem"))
	if err != nil {
		t.Fatalf("expected private.pem to exist: %v", err)
	}
	if privInfo.Size() == 0 {
		t.Fatal("expected non-empty private.pem")
	}
	if perm := privInfo.Mode().Perm(); perm != 0o600 {
		t.Fatalf("expected private.pem mode 0600, got %04o", perm)
	}

	pubInfo, err := os.Stat(filepath.Join(outDir, "public.pem"))
	if err != nil {
		t.Fatalf("expected public.pem to exist: %v", err)
	}
	if pubInfo.Size() == 0 {
		t.Fatal("expected non-empty public.pem")
	}

	authority, err := domain.LoadTokenAuthorityFromDir(outDir, "test-kid")
	if err != nil {
		t.Fatalf("LoadTokenAuthorityFromDir failed: %v", err)
	}
	if authority.KID() != "test-kid" {
		t.Fatalf("expected KID 'test-kid', got %q", authority.KID())
	}
}
