package env

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/FacileStudio/Sonde/apps/api/internal/retention"
	"github.com/FacileStudio/porte"
)

type Config struct {
	DatabaseURL        string
	Port               string
	CORSAllowedOrigins []string
	LogLevel           string
	JournalURL         string
	JournalToken       string
	RawRetentionDays   int
	AllowRegistration  bool
	Porte              porte.Config
}

func Load() (Config, error) {
	env := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Port:        valueOrDefault("PORT", "4000"),
		LogLevel:    valueOrDefault("LOG_LEVEL", "info"),
		CORSAllowedOrigins: csvOrDefault("CORS_ALLOWED_ORIGINS", []string{
			"http://localhost:3000",
			"http://127.0.0.1:3000",
			"http://localhost:5173",
			"http://127.0.0.1:5173",
		}),
		JournalURL:       os.Getenv("JOURNAL_URL"),
		JournalToken:     os.Getenv("JOURNAL_TOKEN"),
		RawRetentionDays: intOrDefault("SONDE_RAW_RETENTION_DAYS", retention.DefaultRawRetentionDays),
	}

	var err error
	if env.AllowRegistration, err = boolOrDefault("ALLOW_REGISTRATION", true); err != nil {
		return Config{}, err
	}

	env.Porte = porte.Config{
		Issuer:       os.Getenv("OIDC_ISSUER"),
		ClientID:     os.Getenv("OIDC_CLIENT_ID"),
		ClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("OIDC_REDIRECT_URL"),
		SuccessURL:   os.Getenv("OIDC_SUCCESS_URL"),
	}
	var ssoOnly bool
	if ssoOnly, err = boolOrDefault("SSO_ONLY", false); err != nil {
		return Config{}, err
	}
	env.Porte.SSOOnly = ssoOnly

	if env.Porte.SSOOnly && !env.Porte.Enabled() {
		return Config{}, fmt.Errorf("SSO_ONLY=true with no OIDC_ISSUER leaves no way to sign in")
	}
	if err := env.Porte.Validate(); err != nil {
		return Config{}, err
	}

	if env.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	port, err := strconv.Atoi(env.Port)
	if err != nil || port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("PORT must be a valid TCP port")
	}
	if err := validateOrigins(env.CORSAllowedOrigins); err != nil {
		return Config{}, err
	}
	if err := validateLogLevel(env.LogLevel); err != nil {
		return Config{}, err
	}

	return env, nil
}

func boolOrDefault(key string, fallback bool) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean", key)
	}
	return parsed, nil
}

func intOrDefault(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}

func valueOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func csvOrDefault(key string, fallback []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return []string{}
	}
	return out
}

func validateOrigins(origins []string) error {
	if len(origins) == 0 {
		return fmt.Errorf("CORS_ALLOWED_ORIGINS must contain at least one origin")
	}

	for _, origin := range origins {
		if origin == "*" {
			continue
		}
		if strings.HasPrefix(origin, "http://") || strings.HasPrefix(origin, "https://") {
			continue
		}
		return fmt.Errorf("CORS_ALLOWED_ORIGINS contains invalid origin %q", origin)
	}

	return nil
}

func validateLogLevel(level string) error {
	switch strings.ToLower(level) {
	case "debug", "info", "warn", "error":
		return nil
	default:
		return fmt.Errorf("LOG_LEVEL must be one of debug, info, warn, error")
	}
}
