package events

import (
	"net/http"
	"strconv"

	"github.com/FacileStudio/GoSvelteBoilerplate/apps/api/internal/middleware"
	"github.com/FacileStudio/GoSvelteBoilerplate/apps/api/modules/auth"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, service *Service, authService *auth.Service) {
	router.Route("/events", func(router chi.Router) {
		router.Get("/", func(w http.ResponseWriter, request *http.Request) {
			resp, err := service.controller.listEvents(request.Context())
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			httpjson.WriteJSON(w, http.StatusOK, resp)
		})

		router.With(middleware.RequireAuth(authService)).Post("/", func(w http.ResponseWriter, request *http.Request) {
			var req CreateRequest
			if err := httpjson.DecodeJSON(w, request, &req); err != nil {
				httpjson.WriteError(w, err)
				return
			}

			resp, err := service.controller.createEvent(request.Context(), &req)
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			httpjson.WriteJSON(w, http.StatusCreated, resp)
		})

		router.Get("/{id}", func(w http.ResponseWriter, request *http.Request) {
			id, err := strconv.ParseInt(chi.URLParam(request, "id"), 10, 64)
			if err != nil {
				httpjson.WriteError(w, errors.Invalid("invalid event id"))
				return
			}

			resp, err := service.controller.getEvent(request.Context(), id)
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			httpjson.WriteJSON(w, http.StatusOK, resp)
		})
	})
}
