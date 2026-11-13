package env

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/FacileStudio/Sonde/apps/api/internal/githubapp"
)

func (env Config) validate() error {
	if err := env.validateIntegrations(); err != nil {
		return err
	}
	return env.validateServer()
}

func (env Config) validateIntegrations() error {
	if os.Getenv("APP_ENV") == "production" && !env.Porte.Enabled() {
		return fmt.Errorf("OIDC_ISSUER is required in production: sign-in goes through Registre only")
	}
	if err := env.Porte.Validate(); err != nil {
		return err
	}
	if env.eventsErr != nil {
		return env.eventsErr
	}
	if env.FleetToken != "" && len(env.FleetToken) < minFleetTokenLength {
		return fmt.Errorf("SONDE_FLEET_TOKEN must be at least %d characters", minFleetTokenLength)
	}
	_, err := githubapp.New(env.GitHubApp)
	return err
}

func (env Config) validateServer() error {
	if env.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	port, err := strconv.Atoi(env.Port)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("PORT must be a valid TCP port")
	}
	if err := validateOrigins(env.CORSAllowedOrigins); err != nil {
		return err
	}
	return validateLogLevel(env.LogLevel)
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
