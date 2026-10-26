package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/yannamsellem/gestion-immo/internal/modules/auth"
	authapp "github.com/yannamsellem/gestion-immo/internal/modules/auth/application"
	"github.com/yannamsellem/gestion-immo/internal/platform/postgres"
	"github.com/yannamsellem/gestion-immo/internal/platform/secret"
)

// Command returns the self-contained CLI command for bootstrapping SUPER_ADMIN and managing user signup approvals.
// It initializes only the PostgreSQL pool and the auth module.
func Command() *cli.Command {
	dbAndKeyFlags := []cli.Flag{
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
	}

	superAdminFlags := append([]cli.Flag{
		&cli.StringFlag{
			Name:     "email",
			Usage:    "SUPER_ADMIN email address",
			Required: true,
			Sources:  cli.EnvVars("SUPER_ADMIN_EMAIL"),
		},
		&cli.StringFlag{
			Name:     "password",
			Usage:    "SUPER_ADMIN password (via --password or SUPER_ADMIN_PASSWORD)",
			Required: true,
			Sources:  cli.EnvVars("SUPER_ADMIN_PASSWORD"),
		},
	}, dbAndKeyFlags...)

	reviewFlags := append([]cli.Flag{
		&cli.StringFlag{
			Name:     "email",
			Usage:    "Email address of the target user account",
			Required: true,
		},
		&cli.StringFlag{
			Name:    "actor-email",
			Usage:   "Optional SUPER_ADMIN email performing the action",
			Sources: cli.EnvVars("SUPER_ADMIN_EMAIL"),
		},
	}, dbAndKeyFlags...)

	return &cli.Command{
		Name:    "bootstrap",
		Aliases: []string{"admin"},
		Usage:   "Bootstrap SUPER_ADMIN configuration and manage user signup approvals",
		Commands: []*cli.Command{
			{
				Name:   "super-admin",
				Usage:  "Create or ensure the SUPER_ADMIN account in APPROVED status",
				Flags:  superAdminFlags,
				Action: bootstrapSuperAdminAction,
			},
			{
				Name:   "approve",
				Usage:  "Approve a PENDING_APPROVAL user account",
				Flags:  reviewFlags,
				Action: reviewUserAction(true),
			},
			{
				Name:   "reject",
				Usage:  "Reject a user account and revoke all active refresh tokens",
				Flags:  reviewFlags,
				Action: reviewUserAction(false),
			},
			{
				Name:  "pending",
				Usage: "List all user accounts currently in PENDING_APPROVAL status",
				Flags: append([]cli.Flag{
					&cli.StringFlag{
						Name:    "actor-email",
						Usage:   "Optional SUPER_ADMIN email performing the query",
						Sources: cli.EnvVars("SUPER_ADMIN_EMAIL"),
					},
				}, dbAndKeyFlags...),
				Action: listPendingAction,
			},
		},
	}
}

func bootstrapSuperAdminAction(ctx context.Context, cmd *cli.Command) error {
	authSvc, cleanup, err := initAuthService(ctx, cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	email := strings.TrimSpace(cmd.String("email"))
	password := strings.TrimSpace(cmd.String("password"))
	userDTO, err := authSvc.BootstrapSuperAdmin(ctx, email, secret.New(password))
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.Root().Writer, "bootstrapped %s user %s (id=%s, status=%s)\n", userDTO.Role, userDTO.Email, userDTO.ID, userDTO.Status)
	return nil
}

func reviewUserAction(approve bool) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		authSvc, cleanup, err := initAuthService(ctx, cmd)
		if err != nil {
			return err
		}
		defer cleanup()

		targetUser, err := authSvc.FindUserByEmail(ctx, strings.TrimSpace(cmd.String("email")))
		if err != nil {
			return err
		}
		actorID, err := authSvc.ResolveSuperAdminID(ctx, strings.TrimSpace(cmd.String("actor-email")))
		if err != nil {
			return err
		}

		var updated authapp.UserDTO
		if approve {
			updated, err = authSvc.ApproveUser(ctx, actorID, targetUser.ID)
		} else {
			updated, err = authSvc.RejectUser(ctx, actorID, targetUser.ID)
		}
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(cmd.Root().Writer, "user %s (id=%s) status=%s\n", updated.Email, updated.ID, updated.Status)
		return nil
	}
}

func listPendingAction(ctx context.Context, cmd *cli.Command) error {
	authSvc, cleanup, err := initAuthService(ctx, cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	actorID, err := authSvc.ResolveSuperAdminID(ctx, strings.TrimSpace(cmd.String("actor-email")))
	if err != nil {
		return err
	}
	users, err := authSvc.ListPendingUsers(ctx, actorID)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(cmd.Root().Writer)
	enc.SetIndent("", "  ")
	return enc.Encode(users)
}

func initAuthService(ctx context.Context, cmd *cli.Command) (*authapp.Service, func(), error) {
	dbURL := secret.New(strings.TrimSpace(cmd.String("database-url")))
	pool, err := postgres.NewPool(ctx, dbURL)
	if err != nil {
		return nil, nil, err
	}
	authMod, err := auth.NewModule(
		pool,
		strings.TrimSpace(cmd.String("jwt-keys-dir")),
		strings.TrimSpace(cmd.String("jwt-kid")),
	)
	if err != nil {
		pool.Close()
		return nil, nil, err
	}
	return authMod.Service(), func() { pool.Close() }, nil
}
