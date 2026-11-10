package fleet

import (
	"net/http"

	"github.com/FacileStudio/tronc/httpjson"
)

// Graph serves every project as a node with the edges the runner derived.
func (h *Handler) Graph(w http.ResponseWriter, r *http.Request) {
	graph, err := h.store.Graph(r.Context())
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, graph)
}

// Conformance serves the per-project check matrix, worst project first.
func (h *Handler) Conformance(w http.ResponseWriter, r *http.Request) {
	matrix, err := h.store.Conformance(r.Context())
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, matrix)
}
