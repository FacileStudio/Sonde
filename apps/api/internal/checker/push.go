package checker

import (
	"net/http"

	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"

	"github.com/go-chi/chi/v5"
)

type heartbeatResponse struct {
	OK bool `json:"ok"`
}

func RegisterHeartbeat(router chi.Router, engine *Engine) {
	router.Post("/push/{token}", func(w http.ResponseWriter, request *http.Request) {
		token := chi.URLParam(request, "token")
		if token == "" || engine == nil {
			httpjson.WriteError(w, errors.NotFound("unknown push token"))
			return
		}
		if err := engine.Heartbeat(request.Context(), token); err != nil {
			httpjson.WriteError(w, err)
			return
		}
		httpjson.WriteJSON(w, http.StatusOK, heartbeatResponse{OK: true})
	})
}
