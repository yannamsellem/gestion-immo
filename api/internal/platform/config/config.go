package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/yannamsellem/gestion-immo/internal/platform/secret"
)

// Config holds the immutable application configuration loaded exclusively from environment variables.
type Config struct {
	DatabaseURL       secret.String
	JWTKeysDir        string
	JWTKID            string
	R2Endpoint        string
	R2AccessKeyID     string
	R2SecretAccessKey secret.String
	R2BucketName      string
	GotenbergURL      string
	EnableOpenBanking bool
	Port              string
}

// LookupFunc matches the signature of os.LookupEnv for dependency-injected configuration loading.
type LookupFunc func(key string) (string, bool)

// Load loads configuration from the process environment (os.LookupEnv).
func Load() (Config, error) {
	return LoadFromEnv(os.LookupEnv)
}

// LoadFromEnv loads and validates all required configuration from the provided lookup function.
func LoadFromEnv(lookup LookupFunc) (Config, error) {
	if lookup == nil {
		return Config{}, fmt.Errorf("config: lookup function must not be nil")
	}

	requiredKeys := []string{
		"DATABASE_URL",
		"R2_ENDPOINT",
		"R2_ACCESS_KEY_ID",
		"R2_SECRET_ACCESS_KEY",
		"R2_BUCKET_NAME",
		"GOTENBERG_URL",
	}

	var missing []string
	values := make(map[string]string, len(requiredKeys))
	for _, key := range requiredKeys {
		val, ok := lookup(key)
		trimmed := strings.TrimSpace(val)
		if !ok || trimmed == "" {
			missing = append(missing, key)
			continue
		}
		values[key] = trimmed
	}

	if len(missing) > 0 {
		return Config{}, fmt.Errorf("config: missing required environment variables: %s", strings.Join(missing, ", "))
	}

	jwtKeysDir := "/app/keys"
	if rawDir, ok := lookup("JWT_KEYS_DIR"); ok && strings.TrimSpace(rawDir) != "" {
		jwtKeysDir = strings.TrimSpace(rawDir)
	}

	jwtKID := ""
	if rawKID, ok := lookup("JWT_KID"); ok && strings.TrimSpace(rawKID) != "" {
		jwtKID = strings.TrimSpace(rawKID)
	}

	port := "8080"
	if rawPort, ok := lookup("PORT"); ok && strings.TrimSpace(rawPort) != "" {
		port = strings.TrimSpace(rawPort)
	}

	enableOpenBanking := false
	if rawOB, ok := lookup("ENABLE_OPEN_BANKING"); ok && strings.TrimSpace(rawOB) != "" {
		parsed, err := strconv.ParseBool(strings.TrimSpace(rawOB))
		if err != nil {
			return Config{}, fmt.Errorf("config: invalid ENABLE_OPEN_BANKING value %q: %w", rawOB, err)
		}
		enableOpenBanking = parsed
	}

	return Config{
		DatabaseURL:       secret.New(values["DATABASE_URL"]),
		JWTKeysDir:        jwtKeysDir,
		JWTKID:            jwtKID,
		R2Endpoint:        values["R2_ENDPOINT"],
		R2AccessKeyID:     values["R2_ACCESS_KEY_ID"],
		R2SecretAccessKey: secret.New(values["R2_SECRET_ACCESS_KEY"]),
		R2BucketName:      values["R2_BUCKET_NAME"],
		GotenbergURL:      values["GOTENBERG_URL"],
		EnableOpenBanking: enableOpenBanking,
		Port:              port,
	}, nil
}

// LogValue implements slog.LogValuer so logging the entire Config struct never leaks secrets.
func (c Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("database_url", c.DatabaseURL),
		slog.String("jwt_keys_dir", c.JWTKeysDir),
		slog.String("jwt_kid", c.JWTKID),
		slog.String("r2_endpoint", c.R2Endpoint),
		slog.String("r2_access_key_id", c.R2AccessKeyID),
		slog.Any("r2_secret_access_key", c.R2SecretAccessKey),
		slog.String("r2_bucket_name", c.R2BucketName),
		slog.String("gotenberg_url", c.GotenbergURL),
		slog.Bool("enable_open_banking", c.EnableOpenBanking),
		slog.String("port", c.Port),
	)
}
