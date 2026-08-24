package auth

import documentation "github.com/FacileStudio/Sonde/apps/api/internal/documentation"

var Documentation = documentation.Module{
	Name:        "auth",
	Description: "Authentication: porte sessions (local password + OIDC single sign-on).",
	Routes: []documentation.Route{
		{
			Method:       "POST",
			Path:         "/auth/register",
			Summary:      "Register a new user",
			Description:  "Creates an account with a local password and returns a session token. Absent when SSO_ONLY=true.",
			Auth:         "public",
			RequestBody:  "RegisterRequest",
			ResponseBody: "AuthResponse",
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid JSON body or invalid registration input."},
				{Status: 409, Code: "already_exists", Description: "A user with the same email already exists."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "POST",
			Path:         "/auth/login",
			Summary:      "Authenticate with email and password",
			Description:  "Issues a session token and sets the session cookie. Absent when SSO_ONLY=true.",
			Auth:         "public",
			RequestBody:  "LoginRequest",
			ResponseBody: "AuthResponse",
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid JSON body or invalid login input."},
				{Status: 401, Code: "unauthenticated", Description: "Email or password is invalid."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "GET",
			Path:         "/auth/config",
			Summary:      "Sign-in configuration for the client",
			Description:  "Reports sso_only and oidc_enabled so the login page knows whether to draw the password form.",
			Auth:         "public",
			ResponseBody: "object",
			Errors:       []documentation.Error{},
		},
		{
			Method:       "POST",
			Path:         "/auth/logout",
			Summary:      "End the current session",
			Description:  "Idempotent: answers the same whether or not the caller had a valid session. Cookie callers must send X-Facile-CSRF.",
			Auth:         "session cookie or bearer token",
			ResponseBody: "object",
			Errors: []documentation.Error{
				{Status: 403, Code: "forbidden", Description: "A cookie-authenticated request without the X-Facile-CSRF header."},
			},
		},
		{
			Method:       "GET",
			Path:         "/auth/oidc",
			Summary:      "Start the OIDC flow",
			Description:  "Redirects to the identity provider with PKCE. Enabled only when OIDC_ISSUER is set.",
			Auth:         "public",
			ResponseBody: "redirect",
			Errors:       []documentation.Error{},
		},
		{
			Method:       "GET",
			Path:         "/auth/oidc/callback",
			Summary:      "OIDC callback",
			Description:  "Exchanges the code, lands the user in the same session as a password login. A failure redirects to the login page with ?error=<reason> — render it outside any SSO_ONLY gate, because under SSO_ONLY this redirect is the only sign-in feedback there is.",
			Auth:         "public",
			ResponseBody: "redirect",
			Errors:       []documentation.Error{},
		},
		{
			Method:       "GET",
			Path:         "/auth/me",
			Summary:      "Return the signed-in user",
			Auth:         "session cookie (with X-Facile-CSRF on writes) or bearer token",
			ResponseBody: "MeResponse",
			Errors: []documentation.Error{
				{Status: 401, Code: "unauthenticated", Description: "No valid session."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
	},
}
