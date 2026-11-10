# Sonde — Architecture

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
4. Open the database, then `migrate.Command` — if the process was invoked as
   `/api migrate <subcommand>`, run it and exit. Otherwise `migrate.Run` applies the embedded
   goose migrations. The `*sql.DB` handle stays for the readiness probe and porte's stores.
5. Start the two background goroutines: `checker.Engine.Run` probes monitors on their
   schedules, `retention.Job.Loop` rolls up and prunes every 24 hours. Both take a context
   cancelled at shutdown.
6. `buildRouter` — porte's session manager, then the OIDC and password kits, then one service
   per package, then the route table.
7. `httpx.NewRouter` — the suite middleware stack, with CORS configured from the environment.
8. `health.Mount(router, health.DB(sqlDB))` — `/health` and `/ready` at both the root and
   under `/api`. `apiref.Mount` puts the API reference at `/docs`.
9. `router.Route("/api", …)` — porte and every domain package register inside this subtree.
10. The SPA catch-all, mounted **last**, and only when `spa.Available(clientDir)` finds a
    built client. Without one, `go run .` still works as a bare API.
11. `http.Server` with explicit timeouts, plus a `SIGINT`/`SIGTERM` graceful shutdown with a
    10-second deadline.

Two ordering rules follow from this and are not negotiable:

- **The SPA catch-all is registered last.** `router.Handle("/*", …)` swallows every path chi
  has not already claimed, so a route added after it is unreachable.
- **Only non-API routes live at the root.** `/health`, `/ready` and `/docs` keep their URLs
  because explicit chi routes beat the catch-all. Everything else goes under `/api`.

## Where the code lives

`apps/api/modules/` holds the two account modules, `auth` and `users`, in the suite's file
shape: `types.go`, `service.go`, `controller.go` or `handler.go`, `router.go`,
`documentation.go`. A module that owns HTTP routes has a `router.go` and nothing else does the
HTTP wiring.

`apps/api/internal/` holds Sonde's own domain, one package per concern:

| Package | Owns |
|---|---|
| `checker` | The probe engine, the HTTP/TCP probes, push heartbeats, and the recorder that writes a check |
| `monitors` | The monitor store and validation |
| `uptime` | Uptime percentages over 24h, 7d, 30d and 90d, read from `checks_hourly` |
| `retention` | The daily rollup and prune job |
| `httpapi` | Monitor CRUD, incidents, config export, status pages, and the public status feed |
| `database` | The GORM connection |
| `env` | Config load and validation |
| `middleware` | `RequireAuth`, which resolves a porte session into an identity |
| `authcontext` | The request-scoped identity |
| `documentation` | The type the API reference registry is assembled from |

`RegisterRoutes` is called from `buildRouter` inside `router.Route("/api", …)`, so a package
declares paths relative to `/api` and never repeats the prefix. Routes that need
authentication wrap with `router.With(requireAuth)`, built once in `buildRouter` from
`middleware.RequireAuth(sessions, authService)`.

`apps/api/apiref_test.go` walks the live router and fails if a mounted route is missing from
the registry, so the reference at `/docs` cannot silently drift from the code.

## Data model

The schema is hand-written SQL in `apps/api/migrations/`, applied in order by `tronc/migrate`.
`00001_sonde.sql` owns the domain, `00002_porte.sql` owns the identity tables porte reads.
The GORM structs in `apps/api/schemas/` mirror those tables for querying; `AutoMigrate` never
runs, so the SQL is the only definition of the schema.

```
users                     monitors
  id            bigint PK   id               bigint PK
  email         text UQ     owner_id         bigint INDEX
  name          text        slug             text UQ
  is_admin      bool        name, target     text
  password_hash text        type             text CHECK (http|tcp|push)
  created_at    tstz        interval_seconds int
                            timeout_seconds  int
                            expected_status  int
                            expected_keyword text
                            enabled          bool
                            push_token       text UQ NULL

checks_raw                checks_hourly / checks_daily
  id             bigint PK   monitor_id     bigint  ┐ PK
  monitor_id     bigint FK   bucket_start   tstz    ┘
  ok             bool        total          bigint
  latency_ms     bigint      failed         bigint
  error          text        avg_latency_ms bigint
  tls_expires_at tstz NULL
  created_at     tstz

incidents                 status_pages / status_page_monitors
  id          bigint PK     id         bigint PK
  monitor_id  bigint FK     slug       text UQ
  opened_at   tstz          title      text
  resolved_at tstz NULL     page_id + monitor_id UQ, position int
  cause       text
```

Every table that hangs off a monitor declares `REFERENCES monitors (id) ON DELETE CASCADE`,
so deleting a monitor removes its checks, rollups, incidents and status-page memberships in
one statement.

`checks_raw` is the only table with an expiry the operator sets. The three tiers and the
storage-limitation reasoning are in [RETENTION.md](RETENTION.md).

## Authentication

Sonde authenticates through [porte](https://github.com/FacileStudio/porte). One
`porte/session` manager owns one sessions table, and both kits issue through it: a password
login via `porte/local` (argon2id) and a federated one via `porte/oidc` (PKCE) produce the
same row, the same `session` cookie and the same logout.

`middleware.RequireAuth` resolves the session, then puts an
`authcontext.Identity{UserID, Email, IsAdmin}` on the request context. Handlers read the
identity from the context rather than re-parsing the header. `httpapi`'s `adminGate` wraps
that middleware to gate the status-page and export routes on `IsAdmin`.

Cookie-authenticated mutating requests need the `X-Facile-CSRF` header; bearer requests are
exempt. Verify client changes with the transport the browser uses, because a bearer-token test
passes while every browser write 403s.

The suite identity provider is **Registre** at `https://sso.facile.studio`, one issuer at the
root. Authentik and `porte.facile.studio` are gone. `porte` the Go library is not: it is what
this section describes. See [AUTH.md](AUTH.md).

## The client

SvelteKit with `adapter-static` and `fallback: 'index.html'`, because the Go binary owns
routing and every route is a client-side one. Runes mode is force-enabled for every
non-`node_modules` file in `svelte.config.js`.

`apps/client/src/lib/backend.ts` calls `/api`, relative, always — never an absolute host,
never a build-time `VITE_API_BASE_URL`. That constant is what forced the old two-container
split, and removing it is what made the one-container shape possible. In development, Vite
proxies `/api` to `http://127.0.0.1:4000`, so local and production take the same path.

UI comes from [`@facile/muse`](https://github.com/FacileStudio/muse), the suite design system,
pinned in `apps/client/package.json` as a GitHub tag. Reach for a muse component before
writing one; local components in `src/lib/components/` exist for what muse does not cover.
There is no test runner configured for the client.

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

`migrate` and `apiref` come from `tronc` too: the first applies the embedded goose migrations,
the second serves the OpenAPI document and its Scalar UI at `/docs`. `apps/api/go.mod` pins
`tronc` at v0.21.0, `tronc/migrate` at v0.1.0 and `porte` at v0.17.0.

`internal/env` is the one deliberate exception to using the chassis: it is local rather than
`troncenv`.

Back to the [documentation index](README.md).
