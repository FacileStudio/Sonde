package projects

import (
	"net/http"

	"github.com/FacileStudio/porte/kit"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts GET /projects behind the session.
func RegisterRoutes(router chi.Router, access *Access, requireAuth func(http.Handler) http.Handler) {
	router.With(requireAuth).Get("/projects", access.list)
}

func (a *Access) list(w http.ResponseWriter, r *http.Request) {
	identity, ok := kit.FromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, errors.Unauthorized("missing auth"))
		return
	}
	refs, err := a.Readable(r.Context(), identity)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, refs)
}
