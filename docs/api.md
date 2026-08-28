# Sonde — API

Every route Sonde serves, read from `apps/api/main.go` and each package's `router.go`.

## Conventions

- Routes live under `/api`. `/health`, `/ready` and `/docs` sit at the root so their
  URLs survive the SPA catch-all; `tronc/health` also mirrors the two probes under `/api`.
- Authentication is a porte session. A browser carries the `session` cookie set by
  `/api/auth/login` or the OIDC callback; a script carries `Authorization: Bearer <token>`.
  Both resolve to the same session row, so logout revokes both. A cookie-authenticated
  mutating request also needs the `X-Facile-CSRF` header; bearer requests are exempt. See
  [AUTH.md](AUTH.md).
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
| Method | Path | Auth | Body | Returns |
|---|---|---|---|---|
| POST | `/api/auth/register` | none | `{email, name, password}` | `{token, user}` |
| POST | `/api/auth/login` | none | `{email, password}` | `{token, user}` |
| GET | `/api/auth/me` | session | — | `{user}` |
| GET | `/api/auth/config` | none | — | `{sso_only, oidc_enabled, allow_registration}` |
| POST | `/api/auth/logout` | session | — | Idempotent; a stale cookie is still cleared |
| GET | `/api/auth/oidc` | none | — | 302 to Registre, PKCE |
| GET | `/api/auth/oidc/callback` | none | — | 302 to the success URL, or to `/login?error=…` |

A `user` is `{id, email, name, is_admin, created_at}`. `register` and `login` are not
registered at all when `SSO_ONLY=true`, so there is no endpoint left to probe for an account.
Everything except `register`, `login` and `me` comes from the porte kits, not from
`modules/auth`. Passwords are argon2id, hashed by `porte/local`.

## Users

| Method | Path | Auth | Body | Returns |
|---|---|---|---|---|
| GET | `/api/users/me` | session | — | `{user}` |
| PATCH | `/api/users/me` | session | `{email?, password?}` | `{user}` |

Both fields on the `PATCH` are optional pointers, so omitting one leaves it alone rather than
clearing it.

## Monitors

| Method | Path | Auth | Returns |
|---|---|---|---|
| GET | `/api/monitors` | session | The caller's monitors |
| POST | `/api/monitors` | session | The created monitor |
| GET | `/api/monitors/{id}` | session | One monitor |
| PATCH | `/api/monitors/{id}` | session | The updated monitor |
| DELETE | `/api/monitors/{id}` | session | Deletes the monitor and its whole history, by cascade |
| GET | `/api/monitors/{id}/incidents` | session | That monitor's incidents |
| GET | `/api/monitors/{id}/uptime` | session | Uptime over 24h, 7d, 30d and 90d |
| GET | `/api/incidents` | session | Incidents across the caller's monitors |

A monitor's `type` is `http`, `tcp` or `push`. `{id}` must be a positive integer; anything
else is a 400 rather than a 404.

An uptime window reports `{window, uptime, total, failed}`, computed from `checks_hourly`. A
window with no data reports a null `uptime` rather than inventing a perfect score.

## Push heartbeats

| Method | Path | Auth | Returns |
|---|---|---|---|
| POST | `/api/push/{token}` | none | `{"ok": true}` |

The token is the monitor's `push_token`, and it is the credential. An unknown token is a 404.

## Status pages and export

| Method | Path | Auth | Returns |
|---|---|---|---|
| GET | `/api/config/export` | admin | JSON snapshot of monitors and status pages, no secrets |
| POST | `/api/status-pages` | admin | The created page |
| GET | `/api/status-pages` | admin | All pages |
| GET | `/api/status-pages/{id}` | admin | One page with its monitors |
| PATCH | `/api/status-pages/{id}` | admin | The updated page |
| DELETE | `/api/status-pages/{id}` | admin | Deletes the page |
| GET | `/api/public/status/{slug}` | none | The public feed the status-page renderer reads |

`admin` means an authenticated caller whose `is_admin` is true; `httpapi.adminGate` returns
403 otherwise. `/api/public/status/{slug}` is deliberately unauthenticated, which is what a
status page is for.

## Adding a route

Inside your package's `router.go`:

```go
func RegisterRoutes(router chi.Router, service *Service, requireAuth func(http.Handler) http.Handler) {
	router.Route("/widgets", func(router chi.Router) {
		router.Get("/", func(w http.ResponseWriter, request *http.Request) {
			resp, err := service.listWidgets(request.Context())
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			httpjson.WriteJSON(w, http.StatusOK, resp)
		})

		router.With(requireAuth).Post("/", service.createWidget)
	})
}
```

Then call `widgets.RegisterRoutes(r, …)` from `buildRouter` **inside**
`router.Route("/api", …)`, and add `widgets.Documentation` to `registry()`. Skipping that
second step fails `apps/api/apiref_test.go`. Paths are relative to `/api`; never repeat the
prefix, or the client ends up calling `/api/api/widgets`, which
`scripts/verify-deploy.sh` checks for on purpose.

Handlers read the caller's identity from the request context through `authcontext`, not by
re-parsing the `Authorization` header.

Back to the [documentation index](README.md).
