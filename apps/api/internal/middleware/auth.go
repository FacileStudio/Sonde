package middleware

import (
	"context"
	"net/http"

	"github.com/FacileStudio/Sonde/apps/api/internal/authcontext"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

type Authenticator interface {
	RequireAuth(http.Handler) http.Handler
}

type IdentityResolver interface {
	IdentityForUser(ctx context.Context, userID int64) (authcontext.Identity, error)
}

// RequireAuth runs behind the session manager's own middleware and hydrates
// what porte deliberately does not carry: porte authenticates the credential
// (cookie or bearer, enforcing X-Facile-CSRF on cookie mutations) and hands on
// a user id; email and is_admin live in this app's users table.
func RequireAuth(sessions Authenticator, resolver IdentityResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return sessions.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			authenticated, ok := porte.From(request.Context())
			if !ok {
				httpjson.WriteError(w, errors.Unauthorized("missing auth token"))
				return
			}
			identity, err := resolver.IdentityForUser(request.Context(), authenticated.UserID)
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			next.ServeHTTP(w, request.WithContext(authcontext.WithIdentity(request.Context(), identity)))
		}))
	}
}
