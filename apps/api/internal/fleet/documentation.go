package fleet

import (
	documentation "github.com/FacileStudio/Sonde/apps/api/internal/documentation"
)

var fleetErrors = []documentation.Error{
	{Status: 401, Code: "unauthenticated", Description: "Authentication is required."},
	{Status: 403, Code: "permission_denied", Description: "The account is not an allow-listed fleet administrator."},
	{Status: 500, Code: "internal", Description: "Unexpected server error."},
}

var Documentation = documentation.Module{
	Name:        "fleet",
	Description: "Fleet maintenance: ranked vulnerability actions, version drift and project state, delivered by a scheduled runner. Restricted to allow-listed administrators.",
	Routes: []documentation.Route{
		{
			Method:       "POST",
			Path:         "/fleet/reports",
			Summary:      "Deliver a fleet report",
			Description:  "Stores one night's scan. The body must be gzip-compressed JSON. Findings resolve only for projects the report marks as covered, and a complete run also resolves projects it no longer lists. Deliveries are serialised and a report not newer than the last one is refused. Disabled until SONDE_FLEET_TOKEN is set.",
			Auth:         "fleet token as a bearer header",
			RequestBody:  Report{},
			ResponseBody: IngestResult{},
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "The body is not valid gzip JSON or fails validation."},
				{Status: 401, Code: "unauthenticated", Description: "The fleet token is wrong."},
				{Status: 404, Code: "not_found", Description: "Ingest is not enabled on this instance."},
				{Status: 409, Code: "already_exists", Description: "A report at least as recent is already stored."},
				{Status: 429, Code: "resource_exhausted", Description: "The report is larger than the limits allow."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "GET",
			Path:         "/fleet/actions",
			Summary:      "List the ranked actions",
			Description:  "Open findings grouped by the upgrade that clears them, ranked act, schedule, then track. The level query narrows the list but the counts always cover all of it.",
			Auth:         "session cookie or bearer token, allow-listed administrator",
			ResponseBody: ActionsResponse{},
			Errors:       fleetErrors,
		},
		{
			Method:       "GET",
			Path:         "/fleet/drift",
			Summary:      "Show version drift",
			Description:  "For each tracked package, the projects that are not on its latest version. Projects under an unexpired hold are marked held.",
			Auth:         "session cookie or bearer token, allow-listed administrator",
			ResponseBody: DriftResponse{},
			Errors:       fleetErrors,
		},
		{
			Method:       "GET",
			Path:         "/fleet/projects",
			Summary:      "List the projects",
			Description:  "Every repository the runner has seen, covered ones first and the most open findings first, with its tier, CI state and open finding count.",
			Auth:         "session cookie or bearer token, allow-listed administrator",
			ResponseBody: []ProjectResponse{},
			Errors:       fleetErrors,
		},
		{
			Method:      "PATCH",
			Path:        "/fleet/projects/{id}",
			Summary:     "Classify a project",
			Description: "Sets the fields people own: tier, version_url and journal_app. Scan data is never changed here.",
			Auth:        "session cookie or bearer token, allow-listed administrator",
			PathParams: []documentation.Field{
				{Name: "id", Type: "integer", Description: "Project id."},
			},
			RequestBody:  ProjectUpdate{},
			ResponseBody: map[string]any{},
			Errors:       append([]documentation.Error{{Status: 404, Code: "not_found", Description: "Project not found."}}, fleetErrors...),
		},
		{
			Method:       "GET",
			Path:         "/fleet/runs",
			Summary:      "List recent runs",
			Description:  "The newest deliveries, with how many repositories were expected and scanned.",
			Auth:         "session cookie or bearer token, allow-listed administrator",
			ResponseBody: []RunResponse{},
			Errors:       fleetErrors,
		},
		{
			Method:       "GET",
			Path:         "/fleet/policies",
			Summary:      "List the holds",
			Description:  "Deliberate holds on a package version, including expired ones.",
			Auth:         "session cookie or bearer token, allow-listed administrator",
			ResponseBody: []Policy{},
			Errors:       fleetErrors,
		},
		{
			Method:       "POST",
			Path:         "/fleet/policies",
			Summary:      "Hold a package on a version",
			Description:  "Requires a reason and an end date within a year.",
			Auth:         "session cookie or bearer token, allow-listed administrator",
			RequestBody:  PolicyInput{},
			ResponseBody: Policy{},
			Errors:       append([]documentation.Error{{Status: 400, Code: "invalid_argument", Description: "A field is missing or the end date is out of range."}}, fleetErrors...),
		},
		{
			Method:      "DELETE",
			Path:        "/fleet/policies/{id}",
			Status:      204,
			Summary:     "Remove a hold",
			Description: "Deletes a hold. The project shows as behind again on the next read.",
			Auth:        "session cookie or bearer token, allow-listed administrator",
			PathParams: []documentation.Field{
				{Name: "id", Type: "integer", Description: "Policy id."},
			},
			Errors: append([]documentation.Error{{Status: 404, Code: "not_found", Description: "Policy not found."}}, fleetErrors...),
		},
	},
}
