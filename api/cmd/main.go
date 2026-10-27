package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/yannamsellem/gestion-immo/cmd/bootstrap"
	"github.com/yannamsellem/gestion-immo/cmd/server"
	"github.com/yannamsellem/gestion-immo/cmd/token"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	app := &cli.Command{
		Name:   "gestion-immo",
		Usage:  "Gestion Immobilière & Comptable — Go Modular Monolith + Streamable HTTP MCP Server",
		Writer: os.Stdout,
		Commands: []*cli.Command{
			server.Command(),
			bootstrap.Command(),
			token.Command(),
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		slog.Error("gestion-immo command failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
