# Sonde

Self-hosted uptime monitoring for the Facile Suite. A Go API probes HTTP endpoints, TCP
ports and push heartbeats, records every result, opens and resolves incidents, and serves
public status pages. A SvelteKit client renders it. Both ship as one container.

This is the suite-native alternative to Uptime Kuma: it reuses porte for auth, Journal for
logs and Antenne for alerts instead of rebuilding a notification zoo, and it prunes its own
check history instead of growing an unbounded table.

## Tech stack

- **API** (`apps/api`): Go 1.26, Chi v5, GORM over PostgreSQL 16, ordered goose migrations
- **Auth**: [`porte`](https://github.com/FacileStudio/porte) v0.17.0, sign-in through Registre
  (OIDC) only, no local passwords
- **Client** (`apps/client`): SvelteKit 2 (`adapter-static`), Svelte 5 runes, Tailwind CSS 4,
  Vite 7, and the [`@facile/muse`](https://github.com/FacileStudio/muse) design system
- **Chassis**: [`tronc`](https://github.com/FacileStudio/tronc) v0.21.0 supplies `errors`,
  `httpjson`, `logger`, `middleware`, `httpx`, `health`, `healthcheck`, `spa`, `apiref`,
  `migrate` and `testdb`. Do not reimplement any of them locally
- **Tooling**: mise (monorepo mode) + bun for the client; dev Postgres via
  Docker, falling back to `PATH` binaries
- **Container**: one multi-stage Dockerfile at the repo root

## Key commands

All commands run from the repo root unless noted.

```sh
mise trust && mise install    # one-time setup: trust config + install toolchains
mise run bootstrap            # install all app dependencies (Go modules + bun)
mise run hooks                # enable the tracked pre-push quality gate
mise run db-up                # start local Postgres on 127.0.0.1:5603
mise run db-down              # stop local Postgres
mise run dev                  # run API (port 4000) + client (port 5173) concurrently
mise run build                # build both apps
mise run check                # gofmt + vet + test (API) + svelte-check (client)
mise run format               # rewrite Go sources in place
```

`scripts/check.sh` is the gate and it is deliberately not invoked through mise: `mise run`
resolves every tool in the merged config before running any task body, so one broken tool in
the user's global config would take the gate down with it. The pre-push hook runs the script
directly for the same reason.

The Go half of the gate also fails on a non-Postgres GORM driver in `go.mod`. Postgres is the
only database in this suite, tests included. `tronc/testdb` gives a test a real one;
`TEST_DATABASE_URL` enables those tests and they skip cleanly without it.

### Per-app tasks

```sh
mise //apps/api:dev           # run API server
mise //apps/api:check         # go fmt + go vet + go test
mise //apps/api:build         # build binary to apps/api/bin/api
mise //apps/client:dev        # Vite dev server
mise //apps/client:check      # svelte-check
docker build -t sonde .       # one image, built from the repo root
```

## Deployment shape (one container)

This is the suite standard. **Do not deviate.**

A deployed app is **one container behind one Traefik router on one hostname**:

```
sonde.facile.studio        -> the Go binary
sonde.facile.studio/api/*  -> the API routes
everything else            -> the built SvelteKit SPA
```

How the pieces fit:

- `main.go` registers every route inside `router.Route("/api", ...)`. Routes that are not API
  and must keep their URL (`/health`, `/ready`, `/docs`) stay at the root: explicit chi routes
  win over the catch-all.
- The SPA is mounted **last**, as `router.Handle("/*", spa.Handler(...))`, using `tronc/spa`.
  `spa.DirFromEnv()` reads `CLIENT_DIR` and falls back to `./client`. `spa.Available()` skips
  the mount when no client was built, so `go run .` still works as a bare API.
- The root `Dockerfile` builds the client with bun, builds the API with Go, and copies both
  into `gcr.io/distroless/static-debian12:nonroot`: binary at `/api`, SPA at `/client`.
  There is **no** `apps/client/Dockerfile` and **no** `apps/api/Dockerfile`.
- `docker-compose.yml` declares a single application service with **no `PathPrefix` and no
  `stripprefix`**. The `/api` prefix must reach the binary intact; stripping it 404s
  everything. The header comment in that file is the canonical explanation. Keep it.
- The container is `nonroot` because it writes nothing. **Anything that mounts a writable
  volume must drop `:nonroot` and the `USER` line**: a named volume is owned by root and a
  non-root process cannot write to it.
- Health checks re-exec the binary (`/api healthcheck`, from `tronc/healthcheck`) because
  distroless has no shell, no `curl` and no `wget`.

Consequences worth internalising: every browser request is same-origin, so CORS is dead
weight in production, cookies stay first-party in Safari, and the client build embeds no
environment-specific API URL. The client calls `/api`, relative, always. See
`apps/client/src/lib/backend.ts`. In development, Vite proxies `/api` to
`http://127.0.0.1:4000` so local and production take the same path.

`scripts/verify-deploy.sh` checks both halves after a deploy. A green `/api/health` says
nothing about what the browser gets.

## Project structure

```
.
├── apps/
│   ├── api/                  # Go API
│   │   ├── internal/         # Sonde's own domain, one package per concern
│   │   │   ├── authcontext/  # Request-scoped identity
│   │   │   ├── checker/      # Probe engine, HTTP/TCP probes, push heartbeats, recorder
│   │   │   ├── database/     # GORM connection
│   │   │   ├── documentation/# The type the API reference registry is built from
│   │   │   ├── env/          # Environment config loader and validation
│   │   │   ├── fleet/        # Fleet maintenance: report ingest, ranked actions, version drift, holds
│   │   │   ├── httpapi/      # Monitors, incidents, export, status pages, public feed
│   │   │   ├── middleware/   # RequireAuth over a porte session
│   │   │   ├── monitors/     # Monitor store and validation
│   │   │   ├── retention/    # Daily rollup + prune job
│   │   │   └── uptime/       # Uptime windows read from checks_hourly
│   │   ├── cmd/fleetscan/    # Nightly fleet runner: GitHub API + douane -> one report to Sonde
│   │   ├── migrations/       # Ordered goose SQL, embedded into the binary
│   │   ├── modules/          # Account modules: auth/, users/
│   │   ├── schemas/          # GORM structs mirroring the SQL. AutoMigrate never runs
│   │   ├── main.go           # Entrypoint: buildRouter, then /api routes, then SPA catch-all
│   │   └── mise.toml
│   └── client/               # SvelteKit frontend (adapter-static -> apps/client/build)
├── scripts/                  # rename.sh / check.sh / postgres-up.sh / postgres-down.sh / verify-deploy.sh
├── Dockerfile                # The only Dockerfile: builds both apps into one image
├── docker-compose.yml        # The canonical deployment shape (read its header comment)
└── mise.toml
```

## Data model

`checks_raw` is one row per probe. `checks_hourly` and `checks_daily` are rollups of
`(monitor_id, bucket_start, total, failed, avg_latency_ms)`. `monitors`, `incidents`,
`status_pages` and `status_page_monitors` are the rest of the domain; `users` and the
`porte_*` tables are identity.

Every table that hangs off a monitor declares `ON DELETE CASCADE`, so deleting a monitor
removes its whole history in one statement.

## Environment variables

Copy `apps/api/.env.example` to `apps/api/.env` for local overrides. Full reference:
`docs/configuration.md`.

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | — | Postgres connection string. The only required variable |
| `PORT` | `4000` | API listen port |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `CORS_ALLOWED_ORIGINS` | four localhost dev origins | Dead weight in production: the client is same-origin |
| `CLIENT_DIR` | `./client` | Where the built SPA lives. Set by the image layout, not by hand |
| `JOURNAL_URL` / `JOURNAL_TOKEN` | empty | Optional log shipping to Journal |
| `SONDE_RAW_RETENTION_DAYS` | `30` | Days of raw checks kept. Fails soft: a bad value silently becomes 30 |
| `SONDE_FLEET_TOKEN` | empty | Token for the fleet runner's report delivery. Empty turns ingest off. At least 32 characters |
| `SONDE_FLEET_ADMINS` | empty | Emails allowed to read fleet data, on top of `is_admin`. Empty closes the read routes |
| `APP_ENV` | `production` in compose | `production` refuses to boot without `OIDC_ISSUER` |
| `OIDC_ISSUER` | empty | `https://sso.facile.studio`. Empty disables SSO |
| `OIDC_CLIENT_ID` | empty | `sonde` |
| `OIDC_CLIENT_SECRET` | empty | Set on the Dokploy service, never committed |
| `OIDC_REDIRECT_URL` | empty | `https://sonde.facile.studio/api/auth/oidc/callback` |
| `OIDC_SUCCESS_URL` | empty | Where a completed login lands |
| `OIDC_CLI_AUDIENCE` | empty | Mounts porte's device-exchange route for the CLI. Production: `facile-cli` |
| `ANTENNE_INSTANCE` | empty | The Antenne instance to publish events to. Empty disables publishing entirely |
| `ANTENNE_APP` | `sonde` | The app name events are attributed to |
| `ANTENNE_SECRET` | empty | Shared secret presented when registering |
| `ANTENNE_INSTANCE_ID` | empty | Optional. Distinguishes two deployments of the same app |

## Conventions

- **Migrations are ordered SQL**, in `apps/api/migrations/*.sql`, applied by `tronc/migrate`
  at boot and embedded in the binary. GORM is a query builder here and `AutoMigrate` never
  runs, so what executes in production is exactly what the tests ran. Add a numbered file with
  both `-- +goose Up` and `-- +goose Down`, and update the struct in `schemas/` by hand.
  `docker run <image> migrate status` answers without starting a server.
- **Routing**: routes register from `buildRouter` **inside `router.Route("/api", ...)`**.
  Anything registered at the root competes with the SPA catch-all, so only put non-API routes
  there (`/health`, `/ready`, `/docs`).
- **Module file shape**: a module under `modules/<name>/` has `types.go`, `service.go`,
  `controller.go` or `handler.go`, `router.go` and `documentation.go`. Domain packages under
  `internal/` follow the same split without the ceremony.
- **Document every route.** `apps/api/apiref_test.go` walks the live router and fails if a
  mounted route is missing from `registry()` in `main.go`, so the reference at `/docs` cannot
  drift from the code.
- **Client API calls**: always relative to `/api` (`apps/client/src/lib/backend.ts`). Never an
  absolute host, never a build-time `VITE_API_BASE_URL`, which is what forced the old
  two-container split.
- **Client UI is muse.** `@facile/muse` is pinned in `apps/client/package.json` as a GitHub
  tag. Reach for a muse component before writing one; `src/lib/components/` is for what muse
  does not cover. Invoke the `muse` skill for frontend work.
- **Client package manager is bun**, with `apps/client/bun.lock` as the only lockfile. The
  Docker client stage runs `bun install --frozen-lockfile`, so changing `package.json` without
  regenerating the lock fails the image build rather than silently drifting.
- **Svelte runes**: runes mode is force-enabled for all non-`node_modules` files in
  `svelte.config.js`. There is no test runner configured for the client.
- **Postgres only.** `scripts/check.sh` fails on a SQLite, MySQL, SQL Server or ClickHouse
  driver in `go.mod`. A test on SQLite builds a different schema from the struct tags and then
  passes, proving nothing about the DDL that ships.
- **Local Postgres**: `scripts/postgres-up.sh` picks a provider automatically: Docker (named
  volume, Postgres 16) if the daemon is up, otherwise `initdb`/`pg_ctl` from `PATH` with data
  in `.local/postgres/`. Runs on port 5603, not 5432, to avoid conflicts.
- **Never put a database in `[tools]`.** `mise run` resolves every tool in the merged config
  before running any task body, so an uninstallable pin disables the whole task runner,
  `mise run check` included. The old `asdf:mise-plugins/mise-postgres@14.22` pin did exactly
  that: it builds from source and fails on macOS.

## Gotchas

- **`is_admin` does not open fleet data.** The first account to sign in becomes admin, and
  erasing the last admin promotes the lowest-numbered user, who under public sign-up can be a
  stranger. Every fleet read route goes through `fleet.Gate`, which also requires the email to
  be in `SONDE_FLEET_ADMINS`. Never guard fleet data with `is_admin` alone. See `docs/FLEET.md`.

- **Auth is porte, through Registre only.** One session manager owns the sessions table and
  `porte/oidc` (PKCE) issues through it. There is no password route and no sign-up page. An
  account created before the cutover links by verified email at its first Facile sign-in, and
  only when it holds no other subject of the same issuer. `porte/directory` polls Registre for
  erased people and `UserStore.EraseAccount` deletes their data (GDPR). Cookie-authenticated
  mutations require `X-Facile-CSRF`; bearer requests are exempt, which is why a bearer-token
  test passes while every browser write 403s. See `docs/AUTH.md`.
- **The identity provider is Registre at `https://sso.facile.studio`, one issuer at the
  root.** Authentik is gone and `porte.facile.studio` 404s. go-oidc compares the discovery
  document's `issuer` byte for byte, so a stale value cannot be aliased into working. Do not
  "fix" references to `github.com/FacileStudio/porte`, the Go library, which is a different
  thing from the dead hostname.
- **A redirect URI without the `/api` prefix fails silently.** porte mounts under `/api`, and
  a bare path does not 404: it lands on the SvelteKit catch-all and returns 200 with HTML, so
  the login just never completes and nothing logs an error.
- **The SPA catch-all must be registered last.** `router.Handle("/*", ...)` swallows every
  path chi has not already claimed, so any route added after it is unreachable.
- **A missing hashed asset 404s instead of returning `index.html`.** `tronc/spa` does that on
  purpose: serving the index for a missing `.js` hides the real error behind a MIME failure.
- **`checks_hourly` is written twice on purpose.** The recorder increments the current hour in
  the same transaction as the raw insert, and the retention job rebuilds every completed hour
  from `checks_raw` with an overwriting upsert. The daily job is what corrects the incremental
  counter. See `docs/RETENTION.md`.
- **Incidents are never pruned.** The retention job touches `checks_raw` and `checks_hourly`
  only. An incident's free-text `cause` outlives the 30-day raw window and goes away only when
  its monitor is deleted.
- `/docs` (the API reference and `/docs/openapi.json`), `/health` and `/ready` sit at the
  root, not under `/api`, so their URLs survive the SPA catch-all. `tronc/health` mirrors the
  two probes under `/api` as well.
