package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/yannamsellem/gestion-immo/cmd/bootstrap"
	"github.com/yannamsellem/gestion-immo/cmd/migrate"
	"github.com/yannamsellem/gestion-immo/cmd/server"
	"github.com/yannamsellem/gestion-immo/cmd/token"
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
	if tokenCmd.Command("create") == nil {
		t.Fatal("token command must expose 'create' subcommand")
	}

	// Verify bootstrap super-admin validates required flags before opening any DB connection
	if err := app.Run(context.Background(), []string{"gestion-immo", "bootstrap", "super-admin", "--email", "yann@example.com"}); err == nil {
		t.Fatal("expected bootstrap super-admin without required flags to fail")
	}
}
