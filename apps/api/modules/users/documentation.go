package users

import documentation "github.com/FacileStudio/Sonde/apps/api/internal/documentation"

var Documentation = documentation.Module{
	Name:        "users",
	Description: "Current-user retrieval and update routes.",
	Routes: []documentation.Route{
		{
			Method:       "GET",
			Path:         "/users/me",
			Summary:      "Return the current user",
			Description:  "Returns the authenticated user derived from the bearer token.",
			Auth:         "bearer token required",
			ResponseBody: MeResponse{},
			Errors: []documentation.Error{
				{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "PATCH",
			Path:         "/users/me",
			Summary:      "Update the current user",
			Description:  "Updates the authenticated user's display name. The email comes from Registre on every sign-in and cannot be changed here; a request carrying it is refused.",
			Auth:         "bearer token required",
			RequestBody:  UpdateRequest{},
			ResponseBody: MeResponse{},
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid JSON body, a missing or empty name, or an attempt to change the email."},
				{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."},
				{Status: 404, Code: "not_found", Description: "The authenticated user no longer exists."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
	},
}
