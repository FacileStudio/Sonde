package tickets

import documentation "github.com/FacileStudio/Sonde/apps/api/internal/documentation"

var Documentation = documentation.Module{
	Name:        "tickets",
	Description: "Ticket generation, validation, and check-in routes.",
	Routes: []documentation.Route{
		{
			Method:      "POST",
			Path:        "/events/{eventID}/tickets",
			Summary:     "Generate a ticket",
			Description: "Creates a ticket for the specified event.",
			Auth:        "bearer token required",
			PathParams: []documentation.Field{
				{Name: "eventID", Type: "int64", Description: "Event identifier."},
			},
			ResponseBody: "GenerateResponse",
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "The event ID path parameter is invalid."},
				{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."},
				{Status: 403, Code: "permission_denied", Description: "The authenticated user does not own the event."},
				{Status: 404, Code: "not_found", Description: "The event does not exist."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "POST",
			Path:         "/tickets/validate",
			Summary:      "Validate a ticket code",
			Description:  "Checks whether a ticket code exists and whether it can still be used.",
			Auth:         "public",
			RequestBody:  "ValidateRequest",
			ResponseBody: "ValidateResponse",
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid JSON body or invalid ticket code input."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "POST",
			Path:         "/tickets/checkin",
			Summary:      "Check in a ticket",
			Description:  "Marks a ticket as used after validating the code and access permissions.",
			Auth:         "bearer token required",
			RequestBody:  "CheckInRequest",
			ResponseBody: "CheckInResponse",
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid JSON body or invalid ticket code input."},
				{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."},
				{Status: 404, Code: "not_found", Description: "The ticket does not exist."},
				{Status: 412, Code: "failed_precondition", Description: "The ticket cannot be checked in in its current state."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
	},
}
