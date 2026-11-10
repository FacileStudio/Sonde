package env

import (
	"os"
	"strconv"
	"strings"

	"github.com/FacileStudio/Sonde/apps/api/internal/events"
	"github.com/FacileStudio/Sonde/apps/api/internal/retention"
	"github.com/FacileStudio/porte"

	antenne "github.com/FacileStudio/antenne-client/go"
)

const minFleetTokenLength = 32

type Config struct {
	DatabaseURL        string
	Port               string
	CORSAllowedOrigins []string
	LogLevel           string
	JournalURL         string
	JournalToken       string
	RawRetentionDays   int
	FleetToken         string
	FleetAdmins        []string
	Porte              porte.Config
	Events             *antenne.Config
	eventsErr          error
}

func Load() (Config, error) {
	env := read()
	if err := env.validate(); err != nil {
		return Config{}, err
	}
	return env, nil
}

func read() Config {
	events, eventsErr := events.ConfigFromEnv()
	return Config{
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
		FleetToken:       os.Getenv("SONDE_FLEET_TOKEN"),
		FleetAdmins:      csvOrDefault("SONDE_FLEET_ADMINS", nil),
		Events:           events,
		eventsErr:        eventsErr,
		Porte: porte.Config{
			Issuer:       os.Getenv("OIDC_ISSUER"),
			ClientID:     os.Getenv("OIDC_CLIENT_ID"),
			ClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("OIDC_REDIRECT_URL"),
			SuccessURL:   os.Getenv("OIDC_SUCCESS_URL"),
			CLIAudience:  os.Getenv("OIDC_CLI_AUDIENCE"),
		},
	}
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
