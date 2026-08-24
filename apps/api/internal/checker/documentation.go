package checker

import documentation "github.com/FacileStudio/Sonde/apps/api/internal/documentation"

var Documentation = documentation.Module{
	Name:        "push",
	Description: "Push heartbeat ingestion. Unauthenticated by design: the token is the credential.",
	Routes: []documentation.Route{
		{
			Method:      "POST",
			Path:        "/push/{token}",
			Summary:     "Record a push heartbeat",
			Description: "Marks the matching push monitor as up. Silence past interval x grace marks it down.",
			Auth:        "public",
			Errors: []documentation.Error{
				{Status: 404, Code: "not_found", Description: "No enabled push monitor matches the token."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
	},
}
