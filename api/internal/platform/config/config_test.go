package config_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/yannamsellem/gestion-immo/internal/platform/config"
)

func TestLoadFromEnv_MissingRequiredSecretFailsAndRedactsLogs(t *testing.T) {
	_, err := config.LoadFromEnv(func(k string) (string, bool) { return "", false })
	if err == nil {
		t.Fatal("expected error when required env vars are missing")
	}

	env := map[string]string{
		"DATABASE_URL":         "postgres://user:supersecretdbpass@postgres:5432/immo?sslmode=disable",
		"JWT_KEYS_DIR":         "/app/keys",
		"JWT_KID":              "kid_prod_2026",
		"R2_ENDPOINT":          "https://acc.r2.cloudflarestorage.com",
		"R2_ACCESS_KEY_ID":     "AKIA123",
		"R2_SECRET_ACCESS_KEY": "r2-secret-value",
		"R2_BUCKET_NAME":       "gestion-immo-files",
		"GOTENBERG_URL":        "http://gotenberg:3000",
		"PORT":                 "8080",
	}
	cfg, err := config.LoadFromEnv(func(k string) (string, bool) { v, ok := env[k]; return v, ok })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.EnableOpenBanking {
		t.Fatal("expected EnableOpenBanking to default to false")
	}
	if cfg.JWTKeysDir != "/app/keys" || cfg.JWTKID != "kid_prod_2026" {
		t.Fatalf("unexpected JWTKeysDir/JWTKID: %q / %q", cfg.JWTKeysDir, cfg.JWTKID)
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	logger.Info("loaded config", "database_url", cfg.DatabaseURL, "r2_secret", cfg.R2SecretAccessKey)
	if strings.Contains(buf.String(), "supersecretdbpass") || strings.Contains(buf.String(), "r2-secret-value") {
		t.Fatalf("secret leaked in logs: %s", buf.String())
	}
}
