package migrate

import (
	"context"
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/yannamsellem/gestion-immo/internal/platform/postgres"
)

// Command returns the self-contained CLI command for running PostgreSQL schema migrations.
func Command() *cli.Command {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:     "database-url",
			Usage:    "PostgreSQL connection URL",
			Required: true,
			Sources:  cli.EnvVars("DATABASE_URL"),
		},
		&cli.StringFlag{
			Name:    "migrations-path",
			Usage:   "Directory containing SQL migration files",
			Value:   "migrations",
			Sources: cli.EnvVars("MIGRATIONS_PATH"),
		},
	}

	return &cli.Command{
		Name:  "migrate",
		Usage: "Run PostgreSQL schema migrations (up or down)",
		Commands: []*cli.Command{
			{
				Name:   "up",
				Usage:  "Apply all pending PostgreSQL migrations",
				Flags:  flags,
				Action: runMigration("up"),
			},
			{
				Name:   "down",
				Usage:  "Roll back the last PostgreSQL migration step",
				Flags:  flags,
				Action: runMigration("down"),
			},
		},
	}
}

func runMigration(direction string) cli.ActionFunc {
	return func(_ context.Context, cmd *cli.Command) error {
		dbURL := strings.TrimSpace(cmd.String("database-url"))
		migPath := strings.TrimSpace(cmd.String("migrations-path"))
		if err := postgres.RunMigrations(dbURL, migPath, direction); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(cmd.Root().Writer, "migrations applied (%s)\n", direction)
		return nil
	}
}
