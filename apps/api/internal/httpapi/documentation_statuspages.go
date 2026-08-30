package httpapi

import (
	"net/http"

	documentation "github.com/FacileStudio/Sonde/apps/api/internal/documentation"
)

var StatusPagesDocumentation = documentation.Module{
	Name:        "status pages",
	Description: "Admin-managed status pages and their unauthenticated public feed.",
	Routes: []documentation.Route{
		{
			Method:       "POST",
			Path:         "/status-pages",
			Summary:      "Create a status page",
			Description:  "monitor_ids sets page membership; array order becomes the display position.",
			Auth:         "administrator role required",
			RequestBody:  StatusPageInput{},
			ResponseBody: StatusPageResponse{},
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid input or an unknown monitor in monitor_ids."},
				{Status: 401, Code: "unauthenticated", Description: "No valid session."},
				{Status: 403, Code: "permission_denied", Description: "The caller is not an administrator."},
				{Status: 409, Code: "already_exists", Description: "A status page with the same slug already exists."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "GET",
			Path:         "/status-pages",
			Summary:      "List status pages with their members in position order",
			Auth:         "administrator role required",
			ResponseBody: []StatusPageResponse{},
			Errors: []documentation.Error{
				{Status: 401, Code: "unauthenticated", Description: "No valid session."},
				{Status: 403, Code: "permission_denied", Description: "The caller is not an administrator."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "GET",
			Path:         "/status-pages/{id}",
			Summary:      "Return one status page",
			PathParams:   []documentation.Field{{Name: "id", Type: "integer", Description: "Status page id."}},
			Auth:         "administrator role required",
			ResponseBody: StatusPageResponse{},
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "id is not a positive integer."},
				{Status: 401, Code: "unauthenticated", Description: "No valid session."},
				{Status: 403, Code: "permission_denied", Description: "The caller is not an administrator."},
				{Status: 404, Code: "not_found", Description: "The status page does not exist."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "PATCH",
			Path:         "/status-pages/{id}",
			Summary:      "Update a status page's title, slug or membership",
			Description:  "monitor_ids replaces the whole membership set; array order becomes the display position.",
			PathParams:   []documentation.Field{{Name: "id", Type: "integer", Description: "Status page id."}},
			Auth:         "administrator role required",
			RequestBody:  StatusPageInput{},
			ResponseBody: StatusPageResponse{},
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid input or an unknown monitor in monitor_ids."},
				{Status: 401, Code: "unauthenticated", Description: "No valid session."},
				{Status: 403, Code: "permission_denied", Description: "The caller is not an administrator."},
				{Status: 404, Code: "not_found", Description: "The status page does not exist."},
				{Status: 409, Code: "already_exists", Description: "A status page with the same slug already exists."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:     "DELETE",
			Path:       "/status-pages/{id}",
			Summary:    "Delete a status page",
			PathParams: []documentation.Field{{Name: "id", Type: "integer", Description: "Status page id."}},
			Auth:       "administrator role required",
			Status:     http.StatusNoContent,
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "id is not a positive integer."},
				{Status: 401, Code: "unauthenticated", Description: "No valid session."},
				{Status: 403, Code: "permission_denied", Description: "The caller is not an administrator."},
				{Status: 404, Code: "not_found", Description: "The status page does not exist."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "GET",
			Path:         "/public/status/{slug}",
			Summary:      "Public status-page data for the client renderer",
			Description:  "Unauthenticated by design. Returns the page title plus per-monitor current status, 24h/7d uptime and open incidents. Never includes owner ids or push tokens.",
			PathParams:   []documentation.Field{{Name: "slug", Type: "string", Description: "Status page slug."}},
			Auth:         "public",
			ResponseBody: PublicStatusPage{},
			Errors: []documentation.Error{
				{Status: 404, Code: "not_found", Description: "No status page with that slug."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
	},
}
