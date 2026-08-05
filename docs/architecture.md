# GoSvelteBoilerplate — Architecture

How one Go binary ends up serving both an API and a SvelteKit client, and the module pattern
every fork is expected to follow.

## Runtime topology

```
Internet ──▶ Traefik ──▶ Go binary (:4000) ──┬──▶ /health, /ready  tronc/health
                                              ├──▶ /docs           generated module docs
                                              ├──▶ /api/*          module routes
                                              └──▶ everything else SPA catch-all (tronc/spa)
                                                                    │
                                                             PostgreSQL 16 (GORM)
```

One container, one Traefik router, one hostname. There is no client container and no second
router. Every browser request is same-origin, which is why CORS is dead weight in production,
cookies stay first-party in Safari, and the client build embeds no environment-specific API
URL.

## Startup sequence

`apps/api/main.go`, in order:

1. `healthcheck.Handle(os.Args)` — if the process was invoked as `/api healthcheck`, answer
   and exit before anything else runs. That is what makes a probe possible in a distroless
   image with no shell and no `curl`.
2. `env.Load()` — read and validate the configuration. `DATABASE_URL` is required; an invalid
   `PORT`, empty CORS list or unknown `LOG_LEVEL` is a startup failure.
3. Build the logger, wrapping it with the Journal SDK handler when both `JOURNAL_URL` and
   `JOURNAL_TOKEN` are set.
4. Open the database, run `schemas.Migrate`, and keep the `*sql.DB` handle for the readiness
   probe.
5. Construct one service per module and assemble the `/docs` payload from each module's
   `Documentation` value.
6. `httpx.NewRouter` — the suite middleware stack, with CORS configured from the environment.
7. `health.Mount(router, health.DB(sqlDB))` — `/health` and `/ready` at both the root and
   under `/api`.
8. `router.Get("/docs", …)` at the root.
9. `router.Route("/api", …)` — every module registers inside this subtree.
10. The SPA catch-all, mounted **last**, and only when `spa.Available(clientDir)` finds a
    built client. Without one, `go run .` still works as a bare API.
11. `http.Server` with explicit timeouts, plus a `SIGINT`/`SIGTERM` graceful shutdown with a
    10-second deadline.

Two ordering rules follow from this and are not negotiable:

- **The SPA catch-all is registered last.** `router.Handle("/*", …)` swallows every path chi
  has not already claimed, so a route added after it is unreachable.
- **Only non-API routes live at the root.** `/health`, `/ready` and `/docs` keep their URLs
  because explicit chi routes beat the catch-all. Everything else goes under `/api`.

## The module pattern

Every feature module in `apps/api/modules/<name>/` has exactly five files:

| File | Holds |
|---|---|
| `types.go` | Request and response structs, with their JSON tags |
| `service.go` | `NewService(db)` and the data access — the only place that touches GORM |
| `controller.go` | Business logic, returning typed responses and `tronc/errors` values |
| `router.go` | `RegisterRoutes(router chi.Router, …)` — HTTP wiring and nothing else |
| `documentation.go` | A `documentation.Module` value, collected into `/docs` |

`RegisterRoutes` is called from `main.go` inside `router.Route("/api", …)`, so a module
declares paths relative to `/api` and never repeats the prefix. Routes that need
authentication wrap with `router.With(middleware.RequireAuth(authService))`.

Follow this shape when adding a module. It is what makes the modules readable next to each
other across thirteen forks.

## Data model

GORM models live in `apps/api/schemas/` and are auto-migrated on every startup by
`schemas.Migrate`.

```
users                        sessions
  id           bigint PK       token       text PK
  email        text UNIQUE     user_id     bigint INDEX
  password_hash text           expires_at  timestamptz
  created_at   timestamptz     created_at  timestamptz

events                       tickets
  id         bigint PK         id         bigint PK
  name       text              event_id   bigint INDEX
  starts_at  timestamptz       code_hash  text UNIQUE
  ends_at    timestamptz NULL  status     text  CHECK (valid|used|revoked)
  owner_id   text INDEX        used_at    timestamptz NULL
  created_at timestamptz       created_at timestamptz
```

`users` and `sessions` are the template. `events` and `tickets` are the **demo domain** —
delete them in a fork, along with `modules/events`, `modules/tickets`,
`internal/ticketcode`, `cmd/seed`, and their registrations in `main.go`.

`Migrate` also adds the `tickets_status_check` constraint through raw SQL wrapped in a
`DO $$ … EXCEPTION WHEN duplicate_object THEN NULL` block, because `AutoMigrate` does not
express check constraints and re-running startup must stay idempotent.

## Authentication

**Not JWT.** `authcrypto.NewToken()` mints 32 random bytes, base64url-encoded. The token is
stored verbatim in the `sessions` table with a 30-day expiry and presented as
`Authorization: Bearer <token>`. There is no signing key and nothing to decode — revocation is
a row delete. Passwords are hashed with argon2id in `internal/authcrypto`.

`middleware.RequireAuth` calls the auth service's `Authenticate`, then puts an
`authcontext.Identity{UserID, Email}` on the request context. Handlers read the identity from
the context rather than re-parsing the header.

The interface is deliberately narrow — `Authenticator` is one method, and the identity payload
is reached through an `interface{ GetEmail() string }` assertion — so a module depends on the
capability, not on the auth package's concrete types.

This is the template's own auth. Apps that federate to Authentik at `porte.facile.studio` add
an OIDC adapter alongside it; see the suite `CLAUDE.md`.

## The client

SvelteKit with `adapter-static` and `fallback: 'index.html'`, because the Go binary owns
routing and every route is a client-side one. Runes mode is force-enabled for every
non-`node_modules` file in `svelte.config.js`.

`apps/client/src/lib/backend.ts` calls `/api`, relative, always — never an absolute host,
never a build-time `VITE_API_BASE_URL`. That constant is what forced the old two-container
split, and removing it is what made the one-container shape possible. In development, Vite
proxies `/api` to `http://127.0.0.1:4000`, so local and production take the same path.

UI components are hand-rolled Svelte 5 in `src/lib/components/ui/` (button, card, input,
label), composed with `class-variance-authority`, `clsx` and `tailwind-merge`. There is no
test runner configured for the client.

## Chassis

Everything cross-cutting comes from `tronc`, not from `internal/`:

| Package | Supplies |
|---|---|
| `httpx` | `NewRouter` — the chi router with the suite middleware stack |
| `middleware` | CORS and request logging |
| `errors` | The suite error values (`Invalid`, `Unauthorized`, `NotFound`, …) |
| `httpjson` | `DecodeJSON`, `WriteJSON`, `WriteError` — the `{"error":{"code","message"}}` envelope |
| `logger` | The structured `slog` logger, wrappable for Journal shipping |
| `health` | `/health` and `/ready`, mounted at both the root and `/api` |
| `healthcheck` | The re-exec probe for distroless images |
| `spa` | The static client handler and `CLIENT_DIR` resolution |

`internal/env` is the one deliberate exception: it is local rather than `troncenv`, and this
template pins `tronc` at 0.6.0.

Back to the [documentation index](README.md).
