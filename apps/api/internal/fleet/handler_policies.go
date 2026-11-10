package fleet

import (
	"net/http"
	"strconv"

	"github.com/FacileStudio/porte/kit"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	policies, err := h.store.Policies(r.Context())
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, policies)
}

func (h *Handler) CreatePolicy(w http.ResponseWriter, r *http.Request) {
	identity, ok := kit.FromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, errors.Unauthorized("unauthorized"))
		return
	}
	var in PolicyInput
	if err := httpjson.DecodeJSON(w, r, &in); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	policy, err := h.store.CreatePolicy(r.Context(), in, identity.Email, h.now())
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusCreated, policy)
}

func (h *Handler) DeletePolicy(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	if err := h.store.DeletePolicy(r.Context(), id); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id < 1 {
		return 0, errors.Invalid("invalid id")
	}
	return id, nil
}
