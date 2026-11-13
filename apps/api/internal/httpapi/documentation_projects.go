package httpapi

import (
	documentation "github.com/FacileStudio/Sonde/apps/api/internal/documentation"
)

// monitorProjectRoute documents linking a monitor to a project. It lives
// beside the monitor routes in the reference, in its own file for length.
var monitorProjectRoute = documentation.Route{
	Method:       "PUT",
	Path:         "/monitors/{id}/project",
	Summary:      "Link a monitor to a project",
	Description:  "Sets the GitHub project the monitor watches, or clears it with project_id null. The caller's GitHub account, linked in Registre, must be able to at least read the repository; fleet administrators may link any project. The monitor's other settings are untouched.",
	PathParams:   []documentation.Field{{Name: "id", Type: "integer", Description: "Monitor id."}},
	Auth:         "session cookie (with X-Facile-CSRF) or bearer token",
	RequestBody:  MonitorProjectInput{},
	ResponseBody: MonitorResponse{},
	Errors: []documentation.Error{
		{Status: 400, Code: "invalid_argument", Description: "Invalid JSON body or id."},
		{Status: 401, Code: "unauthenticated", Description: "No valid session."},
		{Status: 403, Code: "permission_denied", Description: "The caller's GitHub account cannot read the repository."},
		{Status: 404, Code: "not_found", Description: "The monitor or the project does not exist, or the monitor belongs to another user."},
		{Status: 412, Code: "failed_precondition", Description: "The GitHub App is not configured, or the caller has no GitHub account linked in Registre."},
		{Status: 503, Code: "unavailable", Description: "GitHub did not answer."},
		{Status: 500, Code: "internal", Description: "Unexpected server error."},
	},
}
