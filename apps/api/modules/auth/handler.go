package auth

import (
	"net/http"
	"time"

	"github.com/FacileStudio/Sonde/apps/api/internal/authcontext"
	"github.com/FacileStudio/Sonde/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

type Handler struct {
	service *Service
}

func newHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	identity, ok := authcontext.IdentityFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, errors.Unauthorized("not authenticated"))
		return
	}

	user, err := h.service.UserByID(r.Context(), identity.UserID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, MeResponse{User: toUserResponse(*user)})
}

func toUserResponse(user schemas.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		IsAdmin:   user.IsAdmin,
		AvatarURL: user.AvatarURL,
		Locale:    user.Locale,
		CreatedAt: user.CreatedAt.UTC().Format(time.RFC3339),
	}
}
