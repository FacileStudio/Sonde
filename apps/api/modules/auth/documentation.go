package auth

import (
	documentation "github.com/FacileStudio/Sonde/apps/api/internal/documentation"
	"github.com/FacileStudio/porte"
)

var Documentation = documentation.Module{
	Name:        "auth",
	Description: "Authentication: porte sessions, signed in through Registre (OIDC single sign-on).",
	Routes: []documentation.Route{
		{
			Method:       "GET",
			Path:         "/auth/config",
			Summary:      "Sign-in configuration for the client",
			Description:  "Reports oidc_enabled and sso_only, both true wherever an issuer is configured.",
			Auth:         "public",
			ResponseBody: porte.ConfigResponse{},
			Errors:       []documentation.Error{},
		},
		{
			Method:       "POST",
			Path:         "/auth/logout",
			Summary:      "End the current session",
			Description:  "Idempotent: answers the same whether or not the caller had a valid session. Cookie callers must send X-Facile-CSRF.",
			Auth:         "session cookie or bearer token",
			ResponseBody: porte.LogoutResponse{},
			Errors: []documentation.Error{
				{Status: 403, Code: "forbidden", Description: "A cookie-authenticated request without the X-Facile-CSRF header."},
			},
		},
		{
			Method:      "GET",
			Path:        "/auth/oidc",
			Summary:     "Start the OIDC flow",
			Description: "Redirects to the identity provider with PKCE. Enabled only when OIDC_ISSUER is set.",
			Auth:        "public",
			Errors:      []documentation.Error{},
		},
		{
			Method:      "GET",
			Path:        "/auth/oidc/callback",
			Summary:     "OIDC callback",
			Description: "Exchanges the code, lands the user in a porte session. A failure redirects to the login page with ?error=<reason>. The login page shows it, because this redirect is the only sign-in feedback there is.",
			Auth:        "public",
			Errors:      []documentation.Error{},
		},
		{
			Method:       "POST",
			Path:         "/auth/oidc/device/exchange",
			Summary:      "Trade a CLI device token for a Sonde session",
			Description:  "Verifies an access token the provider's device grant issued to the CLI and mints a Sonde session for the matching identity. Mounted only when OIDC_CLI_AUDIENCE is set; absent otherwise, which the CLI reads as a 404 and falls back to the loopback flow. No account is ever created here.",
			Auth:         "public",
			RequestBody:  porte.DeviceExchangeRequest{},
			ResponseBody: porte.ExchangeResponse{},
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid JSON body or a missing access_token."},
				{Status: 401, Code: "unauthenticated", Description: "The token failed verification, or its subject holds no account here."},
			},
		},
		{
			Method:       "GET",
			Path:         "/auth/me",
			Summary:      "Return the signed-in user",
			Auth:         "session cookie (with X-Facile-CSRF on writes) or bearer token",
			ResponseBody: MeResponse{},
			Errors: []documentation.Error{
				{Status: 401, Code: "unauthenticated", Description: "No valid session."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
	},
}
