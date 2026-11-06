package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts the routes porte does not own: GET /auth/me.
//
// /auth/config, /auth/logout and the whole OIDC flow come from the porte kits,
// and sign-in goes through Registre only, so no credential route exists to
// probe for an account.
func RegisterRoutes(router chi.Router, service *Service, requireAuth func(http.Handler) http.Handler) {
	handler := newHandler(service)
	router.With(requireAuth).Get("/auth/me", handler.me)
}
