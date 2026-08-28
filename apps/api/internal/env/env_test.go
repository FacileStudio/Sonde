package env

import (
	"testing"
)

// keys is every variable Load reads. A test clears the lot before setting its
// own, so a value left in the developer's shell cannot decide the outcome.
var keys = []string{
	"DATABASE_URL",
	"PORT",
	"LOG_LEVEL",
	"CORS_ALLOWED_ORIGINS",
	"JOURNAL_URL",
	"JOURNAL_TOKEN",
	"SONDE_RAW_RETENTION_DAYS",
	"ALLOW_REGISTRATION",
	"SSO_ONLY",
	"OIDC_ISSUER",
	"OIDC_CLIENT_ID",
	"OIDC_CLIENT_SECRET",
	"OIDC_REDIRECT_URL",
	"OIDC_SUCCESS_URL",
	"OIDC_CLI_AUDIENCE",
	"ANTENNE_INSTANCE",
	"ANTENNE_APP",
	"ANTENNE_SECRET",
	"ANTENNE_INSTANCE_ID",
}

const databaseURL = "postgres://postgres:postgres@127.0.0.1:5432/sonde?sslmode=disable"

func TestLoad(t *testing.T) {
	cases := []struct {
		name    string
		env     map[string]string
		wantErr bool
	}{
		{
			name: "a database url is the only thing required",
			env:  map[string]string{"DATABASE_URL": databaseURL},
		},
		{
			name:    "no database url",
			env:     map[string]string{},
			wantErr: true,
		},
		{
			// An instance without a secret registers with an empty
			// credential and is refused by the bus, so events look
			// configured and silently publish nothing.
			name: "an antenne instance without a secret",
			env: map[string]string{
				"DATABASE_URL":     databaseURL,
				"ANTENNE_INSTANCE": "https://antenne.facile.studio",
			},
			wantErr: true,
		},
		{
			name: "an antenne instance with a secret",
			env: map[string]string{
				"DATABASE_URL":     databaseURL,
				"ANTENNE_INSTANCE": "https://antenne.facile.studio",
				"ANTENNE_SECRET":   "shared",
			},
		},
		{
			name: "a secret without an instance leaves events off",
			env: map[string]string{
				"DATABASE_URL":   databaseURL,
				"ANTENNE_SECRET": "shared",
			},
		},
		{
			name: "sso only without an issuer",
			env: map[string]string{
				"DATABASE_URL": databaseURL,
				"SSO_ONLY":     "true",
			},
			wantErr: true,
		},
		{
			name: "a port that is not a port",
			env: map[string]string{
				"DATABASE_URL": databaseURL,
				"PORT":         "70000",
			},
			wantErr: true,
		},
		{
			name: "an unknown log level",
			env: map[string]string{
				"DATABASE_URL": databaseURL,
				"LOG_LEVEL":    "chatty",
			},
			wantErr: true,
		},
		{
			name: "an origin with no scheme",
			env: map[string]string{
				"DATABASE_URL":         databaseURL,
				"CORS_ALLOWED_ORIGINS": "sonde.facile.studio",
			},
			wantErr: true,
		},
		{
			name: "a boolean that is not a boolean",
			env: map[string]string{
				"DATABASE_URL":       databaseURL,
				"ALLOW_REGISTRATION": "yes please",
			},
			wantErr: true,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			for _, key := range keys {
				t.Setenv(key, "")
			}
			for key, value := range testCase.env {
				t.Setenv(key, value)
			}

			config, err := Load()
			if testCase.wantErr {
				if err == nil {
					t.Fatal("Load() = nil error, want one")
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() = %v, want nil", err)
			}
			if config.DatabaseURL != databaseURL {
				t.Errorf("DatabaseURL = %q, want %q", config.DatabaseURL, databaseURL)
			}
		})
	}
}

func TestEventsAreDisabledWithoutAnInstance(t *testing.T) {
	for _, key := range keys {
		t.Setenv(key, "")
	}
	t.Setenv("DATABASE_URL", databaseURL)

	config, err := Load()
	if err != nil {
		t.Fatalf("Load() = %v, want nil", err)
	}
	if config.Events.Enabled() {
		t.Error("events are enabled with no ANTENNE_INSTANCE")
	}
}
