# GoSvelteBoilerplate — API

Every route the template serves, read from `apps/api/main.go` and each module's `router.go`.
Half of it is the demo domain and is meant to be deleted.

## Conventions

- Module routes live under `/api`. `/health`, `/ready` and `/docs` sit at the root so their
  URLs survive the SPA catch-all; `tronc/health` also mirrors the two probes under `/api`.
- Authentication is `Authorization: Bearer <token>`, where the token comes from
  `/api/auth/register` or `/api/auth/login`. It is 32 random bytes, base64url-encoded, stored
  verbatim in the `sessions` table with a 30-day expiry. Nothing to decode, nothing to verify
  — revocation is a row delete.
- Errors are `{"error":{"code":"…","message":"…"}}`, written by `tronc/httpjson` from
  `tronc/errors` values.
- **Auth** below is `none` or `bearer`.

## Root

| Method | Path | Auth | Returns |
|---|---|---|---|
| GET | `/health` | none | `{"status":"ok"}` — touches no dependency |
| GET | `/ready` | none | `{"status":"ready"}`, or 503 `{"status":"not_ready"}` if the database ping fails |
| GET | `/api/health` | none | Same as `/health` |
| GET | `/api/ready` | none | Same as `/ready` |
| GET | `/docs` | none | The API reference — Scalar UI over the generated OpenAPI document |
| GET | `/docs/openapi.json` | none | The OpenAPI 3.1 document itself |

Both are served by `tronc/apiref`, mounted from `main.go` on the root router. The document is
generated from the route registry `registry()` assembles out of each module's exported
`Documentation` value, so a new module documents itself by declaring one in `documentation.go`
and being added to that list. `apps/api/apiref_test.go` walks the live router and fails if a
mounted route is missing from the registry, so the reference cannot silently drift from the
code.

## Auth

| Method | Path | Auth | Body | Returns |
|---|---|---|---|---|
| POST | `/api/auth/register` | none | `{email, password}` | 201 `{user_id, token}` |
| POST | `/api/auth/login` | none | `{email, password}` | 200 `{user_id, token}` |

Passwords are hashed with argon2id in `internal/authcrypto`. There is no logout route in the
template — deleting the session row is what revokes a token.

## Users

| Method | Path | Auth | Body | Returns |
|---|---|---|---|---|
| GET | `/api/users/me` | bearer | — | `{user: {id, email}}` |
| PATCH | `/api/users/me` | bearer | `{email?, password?}` | `{user: {id, email}}` |

Both fields on the `PATCH` are optional pointers, so omitting one leaves it alone rather than
clearing it.

## Demo domain — delete in a fork

`modules/events`, `modules/tickets`, `internal/ticketcode`, `cmd/seed`, `schemas/event.go` and
`schemas/ticket.go` exist to show the module pattern with something less trivial than a user
record. They are not template features.

### Events

| Method | Path | Auth | Body | Returns |
|---|---|---|---|---|
| GET | `/api/events` | none | — | `{events: [{id, name, starts, ends, owner_id}]}` |
| POST | `/api/events` | bearer | `{name, starts, ends}` | 201 `{id}` |
| GET | `/api/events/{id}` | none | — | The event. `400` when `{id}` is not an integer |

### Tickets

| Method | Path | Auth | Body | Returns |
|---|---|---|---|---|
| POST | `/api/events/{eventID}/tickets` | bearer | — | 201 `{code, ticket}` |
| POST | `/api/tickets/validate` | none | `{code}` | `{valid, status, ticket}` |
| POST | `/api/tickets/checkin` | bearer | `{code}` | `{ticket}` |

A `ticket` is `{id, event_id, status, used_at?}` with `status` in `valid`, `used`, `revoked` —
enforced by a database check constraint, not only by the application.

The plaintext code is returned once, at generation. Only `code_hash` is stored, which is why
`validate` takes the code rather than an id. Note that generating a ticket is mounted at
`/api/events/{eventID}/tickets` rather than inside the `/tickets` subtree — the module owns
the resource, not the URL prefix.

## Adding a route

Inside your module's `router.go`:

```go
func RegisterRoutes(router chi.Router, service *Service, authService *auth.Service) {
	router.Route("/widgets", func(router chi.Router) {
		router.Get("/", func(w http.ResponseWriter, request *http.Request) {
			resp, err := service.controller.listWidgets(request.Context())
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			httpjson.WriteJSON(w, http.StatusOK, resp)
		})

		router.With(middleware.RequireAuth(authService)).Post("/", createWidget)
	})
}
```

Then call `widgets.RegisterRoutes(r, …)` from `main.go` **inside** `router.Route("/api", …)`,
and add `widgets.Documentation` to `registry()`. Skipping that second step fails
`TestEveryRouteIsDocumented`. Paths are relative to `/api` — never
repeat the prefix, or the client ends up calling `/api/api/widgets`, which
`scripts/verify-deploy.sh` checks for on purpose.

Handlers read the caller's identity from the request context through `authcontext`, not by
re-parsing the `Authorization` header.

Back to the [documentation index](README.md).
