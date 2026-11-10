package webhooks

import (
	"net/http"
	"strconv"

	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/porte/kit"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	store      *Store
	dispatcher *Dispatcher
}

func NewHandler(store *Store, dispatcher *Dispatcher) *Handler {
	return &Handler{store: store, dispatcher: dispatcher}
}

func (h *Handler) List(w http.ResponseWriter, req *http.Request) {
	identity, ok := kit.FromContext(req.Context())
	if !ok {
		httpjson.WriteError(w, errors.Unauthorized("unauthorized"))
		return
	}
	rows, err := h.store.List(req.Context(), identity.UserID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	resp := make([]WebhookResponse, 0, len(rows))
	for _, row := range rows {
		resp = append(resp, toResponse(row))
	}
	httpjson.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) Create(w http.ResponseWriter, req *http.Request) {
	identity, ok := kit.FromContext(req.Context())
	if !ok {
		httpjson.WriteError(w, errors.Unauthorized("unauthorized"))
		return
	}
	var in Input
	if err := httpjson.DecodeJSON(w, req, &in); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	created, err := h.store.Create(req.Context(), identity.UserID, in)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusCreated, toResponse(*created))
}

func (h *Handler) Get(w http.ResponseWriter, req *http.Request) {
	identity, ok := kit.FromContext(req.Context())
	if !ok {
		httpjson.WriteError(w, errors.Unauthorized("unauthorized"))
		return
	}
	id, err := parseID(req)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	row, err := h.store.Get(req.Context(), identity.UserID, id)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, toResponse(*row))
}

func (h *Handler) Update(w http.ResponseWriter, req *http.Request) {
	identity, ok := kit.FromContext(req.Context())
	if !ok {
		httpjson.WriteError(w, errors.Unauthorized("unauthorized"))
		return
	}
	id, err := parseID(req)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	var in Input
	if err := httpjson.DecodeJSON(w, req, &in); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	updated, err := h.store.Update(req.Context(), identity.UserID, id, in)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, toResponse(*updated))
}

func (h *Handler) Delete(w http.ResponseWriter, req *http.Request) {
	identity, ok := kit.FromContext(req.Context())
	if !ok {
		httpjson.WriteError(w, errors.Unauthorized("unauthorized"))
		return
	}
	id, err := parseID(req)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	if err := h.store.Delete(req.Context(), identity.UserID, id); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) TestDirect(w http.ResponseWriter, req *http.Request) {
	var in Input
	if err := httpjson.DecodeJSON(w, req, &in); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	in.normalize()
	if err := in.Validate(); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	result, err := h.dispatcher.Test(req.Context(), schemas.Webhook{
		URL:          in.URL,
		Secret:       in.Secret,
		SecretHeader: in.SecretHeader,
	})
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) TestSaved(w http.ResponseWriter, req *http.Request) {
	identity, ok := kit.FromContext(req.Context())
	if !ok {
		httpjson.WriteError(w, errors.Unauthorized("unauthorized"))
		return
	}
	id, err := parseID(req)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	row, err := h.store.Get(req.Context(), identity.UserID, id)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	result, err := h.dispatcher.Test(req.Context(), *row)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, result)
}

func parseID(req *http.Request) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(req, "id"), 10, 64)
	if err != nil {
		return 0, errors.Invalid("invalid id")
	}
	return id, nil
}
