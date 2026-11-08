package fleet

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/FacileStudio/Sonde/apps/api/internal/authcontext"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

// Config is what the fleet routes read from the environment. An empty Token
// turns report ingest off, and an empty Admins list closes every read route.
type Config struct {
	Token  string
	Admins []string
}

func allowed(admins []string, email string) bool {
	want := strings.ToLower(strings.TrimSpace(email))
	if want == "" {
		return false
	}
	for _, admin := range admins {
		if strings.ToLower(strings.TrimSpace(admin)) == want {
			return true
		}
	}
	return false
}

// Gate lets through a signed-in administrator who is also on the allow-list.
// is_admin alone is not enough: the first account to sign in becomes admin, and
// erasing the last admin promotes the lowest-numbered user, who under public
// sign-up can be a stranger. Fleet data is a ranked list of weaknesses and must
// not follow that promotion.
func Gate(requireAuth func(http.Handler) http.Handler, admins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, ok := authcontext.IdentityFromContext(r.Context())
			if !ok || !identity.IsAdmin || !allowed(admins, identity.Email) {
				httpjson.WriteError(w, errors.Forbidden("fleet access required"))
				return
			}
			next.ServeHTTP(w, r)
		}))
	}
}

func bearer(r *http.Request) string {
	header := r.Header.Get("Authorization")
	scheme, token, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}

func sameToken(got, want string) bool {
	a, b := sha256.Sum256([]byte(got)), sha256.Sum256([]byte(want))
	return subtle.ConstantTimeCompare(a[:], b[:]) == 1
}

// IngestAuth accepts the runner's shared token. It hashes both sides before
// comparing so the check takes the same time whatever the token's length.
func IngestAuth(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token == "" {
				httpjson.WriteError(w, errors.NotFound("fleet ingest is not enabled"))
				return
			}
			if !sameToken(bearer(r), token) {
				httpjson.WriteError(w, errors.Unauthorized("invalid fleet token"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
