package projects

import (
	documentation "github.com/FacileStudio/Sonde/apps/api/internal/documentation"
)

// Documentation registers the projects route in the API reference.
var Documentation = documentation.Module{
	Name:        "projects",
	Description: "The FacileStudio GitHub repositories a monitor can be linked to. Access follows GitHub: a repository is listed when the caller's GitHub account, linked in Registre, can at least read it.",
	Routes: []documentation.Route{
		{
			Method:       "GET",
			Path:         "/projects",
			Summary:      "List the projects the caller may link",
			Description:  "Every project the fleet scan knows, by name, kept to those the caller's GitHub account can read. Fleet administrators see all of them. Answers are cached ten minutes per account and repository.",
			Auth:         "session cookie or bearer token",
			ResponseBody: []Ref{},
			Errors: []documentation.Error{
				{Status: 401, Code: "unauthenticated", Description: "No valid session."},
				{Status: 412, Code: "failed_precondition", Description: "The GitHub App is not configured, or the caller has no GitHub account linked in Registre."},
				{Status: 503, Code: "unavailable", Description: "GitHub did not answer."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
	},
}
