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
			ResponseBody: "MeResponse",
			Errors: []documentation.Error{
				{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "PATCH",
			Path:         "/users/me",
			Summary:      "Update the current user",
			Description:  "Updates the authenticated user's email and/or password.",
			Auth:         "bearer token required",
			RequestBody:  "UpdateRequest",
			ResponseBody: "MeResponse",
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid JSON body or invalid update input."},
				{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."},
				{Status: 404, Code: "not_found", Description: "The authenticated user no longer exists."},
				{Status: 409, Code: "already_exists", Description: "A user with the same email already exists."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
	},
}
