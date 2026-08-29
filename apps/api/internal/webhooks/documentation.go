package webhooks

import documentation "github.com/FacileStudio/Sonde/apps/api/internal/documentation"

var Documentation = documentation.Module{
	Name:        "webhooks",
	Description: "Webhook management and delivery configuration for outage and recovery alerts.",
	Routes: []documentation.Route{
		{
			Method:       "GET",
			Path:         "/webhooks",
			Summary:      "List configured webhooks",
			Description:  "Returns all webhook endpoints configured by the authenticated user.",
			Auth:         "bearer token required",
			ResponseBody: "[]WebhookResponse",
			Errors: []documentation.Error{
				{Status: 401, Code: "unauthorized", Description: "Authentication is required."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "POST",
			Path:         "/webhooks",
			Summary:      "Create a webhook",
			Description:  "Registers a new webhook destination for monitor outage and recovery events.",
			Auth:         "bearer token required",
			RequestBody:  "Input",
			ResponseBody: "WebhookResponse",
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid webhook configuration or URL."},
				{Status: 401, Code: "unauthorized", Description: "Authentication is required."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "POST",
			Path:         "/webhooks/test",
			Summary:      "Test a webhook configuration",
			Description:  "Sends a test alert payload to an unsaved webhook URL and returns the outcome.",
			Auth:         "bearer token required",
			RequestBody:  "Input",
			ResponseBody: "TestResult",
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid URL."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "GET",
			Path:         "/webhooks/{id}",
			Summary:      "Get a webhook",
			Description:  "Retrieves a single webhook by ID.",
			Auth:         "bearer token required",
			ResponseBody: "WebhookResponse",
			Errors: []documentation.Error{
				{Status: 401, Code: "unauthorized", Description: "Authentication is required."},
				{Status: 404, Code: "not_found", Description: "Webhook not found."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "PATCH",
			Path:         "/webhooks/{id}",
			Summary:      "Update a webhook",
			Description:  "Updates URL, secret or enabled status for a webhook.",
			Auth:         "bearer token required",
			RequestBody:  "Input",
			ResponseBody: "WebhookResponse",
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Invalid input."},
				{Status: 401, Code: "unauthorized", Description: "Authentication is required."},
				{Status: 404, Code: "not_found", Description: "Webhook not found."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:      "DELETE",
			Path:        "/webhooks/{id}",
			Summary:     "Delete a webhook",
			Description: "Removes a webhook destination.",
			Auth:        "bearer token required",
			Errors: []documentation.Error{
				{Status: 401, Code: "unauthorized", Description: "Authentication is required."},
				{Status: 404, Code: "not_found", Description: "Webhook not found."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
		{
			Method:       "POST",
			Path:         "/webhooks/{id}/test",
			Summary:      "Test a saved webhook",
			Description:  "Sends a test alert payload to an existing webhook destination.",
			Auth:         "bearer token required",
			ResponseBody: "TestResult",
			Errors: []documentation.Error{
				{Status: 401, Code: "unauthorized", Description: "Authentication is required."},
				{Status: 404, Code: "not_found", Description: "Webhook not found."},
				{Status: 500, Code: "internal", Description: "Unexpected server error."},
			},
		},
	},
}
