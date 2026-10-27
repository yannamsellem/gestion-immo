package token

import (
	"context"
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/yannamsellem/gestion-immo/internal/modules/auth"
	"github.com/yannamsellem/gestion-immo/internal/platform/postgres"
	"github.com/yannamsellem/gestion-immo/internal/platform/secret"
)

// Command returns the self-contained CLI command for managing Antigravity MCP Bearer tokens.
// It initializes only the PostgreSQL pool and the auth module.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "token",
		Usage: "Manage Antigravity MCP Bearer tokens",
		Commands: []*cli.Command{
			{
				Name:  "create",
				Usage: "Generate a new SHA-256 hashed MCP Bearer token for an approved user",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "database-url",
						Usage:    "PostgreSQL connection URL",
						Required: true,
						Sources:  cli.EnvVars("DATABASE_URL"),
					},
					&cli.StringFlag{
						Name:    "jwt-keys-dir",
						Usage:   "Directory containing Ed25519 JWT keys",
						Value:   "/app/keys",
						Sources: cli.EnvVars("JWT_KEYS_DIR"),
					},
					&cli.StringFlag{
						Name:    "jwt-kid",
						Usage:   "Active JWT key identifier (kid)",
						Sources: cli.EnvVars("JWT_KID"),
					},
					&cli.StringFlag{
						Name:     "email",
						Usage:    "Approved user email address",
						Required: true,
						Sources:  cli.EnvVars("SUPER_ADMIN_EMAIL"),
					},
					&cli.StringFlag{
						Name:    "label",
						Usage:   "Descriptive token label (e.g. antigravity-mcp)",
						Value:   "antigravity-mcp",
						Sources: cli.EnvVars("MCP_TOKEN_LABEL"),
					},
				},
				Action: createMCPTokenAction,
			},
		},
	}
}

func createMCPTokenAction(ctx context.Context, cmd *cli.Command) error {
	dbURL := secret.New(strings.TrimSpace(cmd.String("database-url")))
	pool, err := postgres.NewPool(ctx, dbURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	authMod, err := auth.NewModule(
		pool,
		strings.TrimSpace(cmd.String("jwt-keys-dir")),
		strings.TrimSpace(cmd.String("jwt-kid")),
	)
	if err != nil {
		return err
	}

	email := strings.TrimSpace(cmd.String("email"))
	label := strings.TrimSpace(cmd.String("label"))
	user, err := authMod.Service().FindUserByEmail(ctx, email)
	if err != nil {
		return err
	}

	rawToken, err := authMod.Service().CreateMCPToken(ctx, user.ID, label)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(cmd.Root().Writer, rawToken.Value())
	return nil
}
