package users

import (
	"net/http"

	"github.com/FacileStudio/GoSvelteBoilerplate/apps/api/internal/middleware"
	"github.com/FacileStudio/GoSvelteBoilerplate/apps/api/modules/auth"
	"github.com/FacileStudio/tronc/httpjson"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, service *Service, authService *auth.Service) {
	router.Route("/users", func(router chi.Router) {
		router.With(middleware.RequireAuth(authService)).Get("/me", func(w http.ResponseWriter, request *http.Request) {
			resp, err := service.controller.me(request.Context())
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			httpjson.WriteJSON(w, http.StatusOK, resp)
		})

		router.With(middleware.RequireAuth(authService)).Patch("/me", func(w http.ResponseWriter, request *http.Request) {
			var req UpdateRequest
			if err := httpjson.DecodeJSON(w, request, &req); err != nil {
				httpjson.WriteError(w, err)
				return
			}

			resp, err := service.controller.updateMe(request.Context(), &req)
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			httpjson.WriteJSON(w, http.StatusOK, resp)
		})
	})
}
