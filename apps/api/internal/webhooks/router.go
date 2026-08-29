package webhooks

import (
	"net/http"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/go-chi/chi/v5"
)

type WebhookResponse struct {
	ID           int64  `json:"id"`
	OwnerID      int64  `json:"owner_id"`
	Name         string `json:"name"`
	URL          string `json:"url"`
	SecretHeader string `json:"secret_header"`
	Enabled      bool   `json:"enabled"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

func toResponse(w schemas.Webhook) WebhookResponse {
	return WebhookResponse{
		ID:           w.ID,
		OwnerID:      w.OwnerID,
		Name:         w.Name,
		URL:          w.URL,
		SecretHeader: w.SecretHeader,
		Enabled:      w.Enabled,
		CreatedAt:    w.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    w.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func RegisterRoutes(router chi.Router, store *Store, dispatcher *Dispatcher, requireAuth func(http.Handler) http.Handler) {
	h := NewHandler(store, dispatcher)

	router.Route("/webhooks", func(r chi.Router) {
		r.Use(requireAuth)

		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Post("/test", h.TestDirect)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.Get)
			r.Patch("/", h.Update)
			r.Delete("/", h.Delete)
			r.Post("/test", h.TestSaved)
		})
	})
}
