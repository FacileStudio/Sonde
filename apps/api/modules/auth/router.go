package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts the routes porte does not own.
//
// /auth/config, /auth/logout and the whole OIDC flow come from the kits; what
// is left here is the local password path and /auth/me. Under SSO_ONLY the
// credential routes are not registered rather than rejected, so there is no
// endpoint left to probe for an account.
func RegisterRoutes(router chi.Router, service *Service, ssoOnly bool, requireAuth func(http.Handler) http.Handler, clearCookie func(http.ResponseWriter, *http.Request)) {
	handler := newHandler(service, clearCookie)
	if !ssoOnly {
		router.Post("/auth/register", handler.register)
		router.Post("/auth/login", handler.login)
	}
	router.With(requireAuth).Get("/auth/me", handler.me)
}
