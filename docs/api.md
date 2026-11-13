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
| GET | `/api/auth/me` | session | — | `{user}` |
| GET | `/api/auth/config` | none | — | `{sso_only, oidc_enabled}` |
| POST | `/api/auth/logout` | session | — | Idempotent; a stale cookie is still cleared |
| GET | `/api/auth/oidc` | none | — | 302 to Registre, PKCE |
| GET | `/api/auth/oidc/callback` | none | — | 302 to the success URL, or to `/login?error=…` |

A `user` is `{id, email, name, is_admin, avatar_url, locale, created_at}`. Sign-in goes through
Registre only: there is no register or login route. Everything except `me` comes from the
porte kits, not from `modules/auth`. `GET /version` answers `{version}` at the root.

## Users

| Method | Path | Auth | Body | Returns |
|---|---|---|---|---|
| GET | `/api/users/me` | session | — | `{user}` |
| PATCH | `/api/users/me` | session | `{name}` | `{user}` |

The `PATCH` changes the display name only. The email comes from Registre on every sign-in, and a
request carrying `email` is refused with 400: the fleet allow-list trusts that address, so a user
must not be able to set it.

## Monitors

| Method | Path | Auth | Returns |
|---|---|---|---|
| GET | `/api/monitors` | session | The caller's monitors |
| POST | `/api/monitors` | session | The created monitor |
| GET | `/api/monitors/{id}` | session | One monitor |
| PATCH | `/api/monitors/{id}` | session | The updated monitor |
| DELETE | `/api/monitors/{id}` | session | Deletes the monitor and its whole history, by cascade |
| PUT | `/api/monitors/{id}/project` | session | Links the monitor to a project (`{"project_id": n}`) or unlinks it (`null`). Needs GitHub read on the repository; 412 without the GitHub App or a linked GitHub account |
| GET | `/api/projects` | session | The projects the caller's GitHub account can read, `[{id, name}]`. Fleet admins see all. 412 without the GitHub App or a linked GitHub account |
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

## Fleet

Fleet maintenance data, delivered by a scheduled runner. Read routes need a signed-in
administrator whose email is in `SONDE_FLEET_ADMINS`; anyone else gets 403. See
[FLEET.md](FLEET.md).

| Method | Path | Auth | Returns |
|---|---|---|---|
| POST | `/api/fleet/reports` | fleet token | Stores one gzip-compressed report. 404 until `SONDE_FLEET_TOKEN` is set; 409 when the report is not newer than the last one stored |
| GET | `/api/fleet/actions` | fleet admin | Open findings grouped by the upgrade that clears them, ranked `act`, `schedule`, `track`. `?level=` narrows the list, never the counts |
| GET | `/api/fleet/drift` | fleet admin | Per tracked package, the projects not on the latest version, with holds marked |
| GET | `/api/fleet/graph` | fleet admin | Every project as a node, and the `lib`, `call`, `event`, `identity` and `logs` edges the runner derived from its files |
| GET | `/api/fleet/conformance` | fleet admin | The check matrix: one row per project, checks that did not apply left out, worst project first |
| GET | `/api/fleet/projects` | fleet admin | Every repository: covered ones first, the most open findings first, each with its linked `monitors` |
| PATCH | `/api/fleet/projects/{id}` | fleet admin | Sets `tier`, `version_url`, `journal_app` |
| GET | `/api/fleet/runs` | fleet admin | The newest deliveries and how complete each was |
| POST | `/api/fleet/scans` | fleet admin | Queues the fleet-scan workflow on GitHub. Once every five minutes; 412 without the GitHub App env, or while the App lacks Actions write |
| GET | `/api/fleet/policies` | fleet admin | Holds, expired ones included |
| POST | `/api/fleet/policies` | fleet admin | Holds a package on a version. Needs a reason and an end within a year |
| DELETE | `/api/fleet/policies/{id}` | fleet admin | Removes a hold |

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

Handlers read the caller's identity from the request context through `kit.FromContext` (`porte/kit`), not by
re-parsing the `Authorization` header.

Back to the [documentation index](README.md).
