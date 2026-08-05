package tickets

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
	router.With(middleware.RequireAuth(authService)).Post("/events/{eventID}/tickets", func(w http.ResponseWriter, request *http.Request) {
		eventID, err := strconv.ParseInt(chi.URLParam(request, "eventID"), 10, 64)
		if err != nil {
			httpjson.WriteError(w, errors.Invalid("invalid event id"))
			return
		}

		resp, err := service.controller.generateTicket(request.Context(), eventID)
		if err != nil {
			httpjson.WriteError(w, err)
			return
		}
		httpjson.WriteJSON(w, http.StatusCreated, resp)
	})

	router.Route("/tickets", func(router chi.Router) {
		router.Post("/validate", func(w http.ResponseWriter, request *http.Request) {
			var req ValidateRequest
			if err := httpjson.DecodeJSON(w, request, &req); err != nil {
				httpjson.WriteError(w, err)
				return
			}

			resp, err := service.controller.validateTicket(request.Context(), &req)
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			httpjson.WriteJSON(w, http.StatusOK, resp)
		})

		router.With(middleware.RequireAuth(authService)).Post("/checkin", func(w http.ResponseWriter, request *http.Request) {
			var req CheckInRequest
			if err := httpjson.DecodeJSON(w, request, &req); err != nil {
				httpjson.WriteError(w, err)
				return
			}

			resp, err := service.controller.checkInTicket(request.Context(), &req)
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			httpjson.WriteJSON(w, http.StatusOK, resp)
		})
	})
}
