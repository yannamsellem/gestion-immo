package token

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/yannamsellem/gestion-immo/internal/modules/auth"
	"github.com/yannamsellem/gestion-immo/internal/modules/auth/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/postgres"
	"github.com/yannamsellem/gestion-immo/internal/platform/secret"
)

// Command returns the self-contained CLI command for managing Antigravity MCP Bearer tokens
// and generating Ed25519 JWT signing keys.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "token",
		Usage: "Manage Antigravity MCP Bearer tokens and Ed25519 JWT keys",
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
			{
				Name:  "keygen",
				Usage: "Generate an Ed25519 JWT signing keypair (private.pem and public.pem)",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "out-dir",
						Usage:   "Output directory for generated PEM files",
						Value:   "./keys",
						Sources: cli.EnvVars("JWT_KEYS_DIR"),
					},
					&cli.StringFlag{
						Name:    "kid",
						Usage:   "Key identifier (kid) for the generated keypair",
						Value:   "default",
						Sources: cli.EnvVars("JWT_KID"),
					},
				},
				Action: keygenAction,
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

func keygenAction(_ context.Context, cmd *cli.Command) error {
	outDir := strings.TrimSpace(cmd.String("out-dir"))
	if outDir == "" {
		outDir = "./keys"
	}
	kid := strings.TrimSpace(cmd.String("kid"))

	authority, err := domain.GenerateTokenAuthority(kid)
	if err != nil {
		return err
	}

	privPEM, pubPEM, err := authority.EncodePEM()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(outDir, 0o700); err != nil {
		return fmt.Errorf("token keygen: create output directory %s: %w", outDir, err)
	}

	privPath := filepath.Join(outDir, "private.pem")
	if err := os.WriteFile(privPath, privPEM, 0o600); err != nil {
		return fmt.Errorf("token keygen: write private key: %w", err)
	}

	pubPath := filepath.Join(outDir, "public.pem")
	if err := os.WriteFile(pubPath, pubPEM, 0o644); err != nil {
		return fmt.Errorf("token keygen: write public key: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.Root().Writer, "generated Ed25519 JWT keys in %s (kid=%s)\n", outDir, authority.KID())
	return nil
}
