package events

import documentation "github.com/FacileStudio/Sonde/apps/api/internal/documentation"

var Documentation = documentation.Module{
	Name:        "events",
	Description: "Event listing, creation, and lookup routes.",
	Routes: []documentation.Route{
		{
			Method:       "GET",
			Path:         "/events",
			Summary:      "List events",
			Description:  "Returns all events ordered by newest first.",
			Auth:         "public",
			ResponseBody: "ListResponse",
			Errors: []documentation.Error{
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "POST",
			Path:         "/events",
			Summary:      "Create an event",
			Description:  "Creates an event owned by the authenticated user.",
			Auth:         "bearer token required",
			RequestBody:  "CreateRequest",
			ResponseBody: "CreateResponse",
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid JSON body or invalid event input."},
				{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:      "GET",
			Path:        "/events/{id}",
			Summary:     "Fetch an event",
			Description: "Returns a single event by ID.",
			Auth:        "public",
			PathParams: []documentation.Field{
				{Name: "id", Type: "int64", Description: "Event identifier."},
			},
			ResponseBody: "Event",
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "The event ID path parameter is invalid."},
				{Status: 404, Code: "not_found", Description: "The event does not exist."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
	},
}
