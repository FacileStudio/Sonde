# GoSvelteBoilerplate

Monorepo boilerplate for a Go API + SvelteKit frontend, managed with mise.

<!-- fork:begin -->
## Forking

A fresh fork inherits the module path `github.com/FacileStudio/GoSvelteBoilerplate/apps/api`,
which is wrong for every project that is not this one. Run `scripts/rename.sh <NewName>` on a
clean tree before doing anything else - it rewrites `go.mod`, every scoped Go import, the
docs, and the dev database name/port. See the README for the full checklist.

<!-- fork:end -->
## Tech Stack

- **API** (`apps/api`): Go 1.24, Chi router, GORM + Postgres, DB-backed session tokens
- **Client** (`apps/client`): SvelteKit (`adapter-static`), Svelte 5 (runes), Tailwind CSS v4, Vite 7, TypeScript
- **Shared chassis**: [`tronc`](https://github.com/FacileStudio/tronc) supplies `env`, `errors`, `httpjson`, `logger`, `middleware`, `httpx`, `health`, `healthcheck` and `spa` — do not reimplement them locally
- **Tooling**: mise (monorepo mode, experimental) + bun for the client; dev Postgres via Docker, falling back to `PATH` binaries
- **Containerization**: one multi-stage Dockerfile at the repo root (bun builds the SPA, Go builds the API, distroless runtime serves both)

## Prerequisites

- [mise](https://mise.jdx.dev/) installed and activated
- `MISE_EXPERIMENTAL=1` if monorepo tasks are not picked up automatically

## Key Commands

All commands run from the repo root unless noted.

```sh
mise trust && mise install    # one-time setup: trust config + install toolchains
mise run bootstrap            # install all app dependencies (Go modules + bun)
mise run db-up                # start local Postgres on 127.0.0.1:5433
mise run db-down              # stop local Postgres
mise run dev                  # run API (port 4000) + client (port 5173) concurrently
mise run build                # build both apps
mise run check                # fmt + vet + test (API) + svelte-check (client)
```

### API-only (from root or from `apps/api`)

```sh
mise //apps/api:dev           # run API server
mise //apps/api:test          # run Go tests
mise //apps/api:check         # go fmt + go vet + go test
mise //apps/api:build         # build binary to apps/api/bin/api
mise //apps/api:tidy          # go mod tidy
```

### Client-only (from root or from `apps/client`)

```sh
mise //apps/client:dev        # Vite dev server
mise //apps/client:build      # production build
mise //apps/client:check      # svelte-check
```

### Docker

```sh
docker build -t gosvelteboilerplate .   # one image, built from the repo root
```

## Deployment shape (one container)

This is the suite standard and the reason forks inherit this repo. **Do not deviate.**

A deployed app is **one container behind one Traefik router on one hostname**:

```
gosvelteboilerplate.facile.studio        -> the Go binary
gosvelteboilerplate.facile.studio/api/*  -> the API routes
everything else                          -> the built SvelteKit SPA
```

How the pieces fit:

- `main.go` registers every application module inside `router.Route("/api", ...)`. Routes
  that are not API and must keep their URL (`/health`, `/ready`, `/docs`) stay at the root:
  explicit chi routes win over the catch-all.
- The SPA is mounted **last**, as `router.Handle("/*", spa.Handler(...))`, using
  `tronc/spa`. `spa.DirFromEnv()` reads `CLIENT_DIR` and falls back to `./client`.
  `spa.Available()` skips the mount when no client was built, so `go run .` still works
  as a bare API.
- The root `Dockerfile` builds the client with bun, builds the API with Go, and copies
  both into `gcr.io/distroless/static-debian12:nonroot` — binary at `/api`, SPA at
  `/client`. There is **no** `apps/client/Dockerfile` and **no** `apps/api/Dockerfile`.
- `docker-compose.yml` declares a single application service with **no `PathPrefix` and no
  `stripprefix`**. The `/api` prefix must reach the binary intact; stripping it 404s
  everything. The header comment in that file is the canonical explanation — keep it.
- The container is `nonroot` because it writes nothing. **A fork that mounts a writable
  volume must drop `:nonroot` and the `USER` line**: a named volume is owned by root and a
  non-root process cannot write to it.
- Health checks re-exec the binary (`/api healthcheck`, from `tronc/healthcheck`) because
  distroless has no shell, no `curl` and no `wget`.

Consequences worth internalising: every browser request is same-origin, so CORS is dead
weight in production, cookies stay first-party in Safari, and the client build embeds no
environment-specific API URL. The client calls `/api`, relative, always — see
`apps/client/src/lib/backend.ts`. In development, Vite proxies `/api` to
`http://127.0.0.1:4000` so local and production take the same path.

## Project Structure

```
.
├── apps/
│   ├── api/                  # Go API
│   │   ├── cmd/seed/         # DB seed script (demo domain)
│   │   ├── internal/         # Shared internal packages (errors, httpjson, logger and the
│   │   │   │                 # CORS/logging middleware now come from tronc, not from here)
│   │   │   ├── authcontext/  # Request-scoped auth context
│   │   │   ├── authcrypto/   # argon2id password hashing + random session tokens
│   │   │   ├── database/     # GORM database connection
│   │   │   ├── documentation/# Auto-docs type definitions
│   │   │   ├── env/          # Environment config loader
│   │   │   ├── middleware/   # Bearer-token auth middleware
│   │   │   └── ticketcode/   # Ticket code generation (demo domain)
│   │   ├── modules/          # Feature modules (controller/service/router/types/docs per module)
│   │   │   ├── auth/
│   │   │   ├── events/        # demo domain
│   │   │   ├── tickets/       # demo domain
│   │   │   └── users/
│   │   ├── schemas/          # GORM models + auto-migration
│   │   ├── main.go           # Entrypoint: /api routes, then the SPA catch-all
│   │   └── mise.toml         # API-specific mise tasks
│   └── client/               # SvelteKit frontend (adapter-static -> apps/client/build)
│       ├── src/
│       │   ├── lib/          # Components (ui/), utils, assets
│       │   └── routes/       # SvelteKit file-based routing
│       ├── bun.lock
│       ├── svelte.config.js
│       └── vite.config.ts
├── scripts/                  # rename.sh / check.sh / postgres-up.sh / postgres-down.sh
├── Dockerfile                # The only Dockerfile: builds both apps into one image
├── docker-compose.yml        # The canonical deployment shape (read its header comment)
├── mise.toml                 # Root mise config (monorepo orchestration)
└── mise.lock
```

## Environment Variables (API)

Copy `apps/api/.env.example` to `apps/api/.env` for local overrides. Defaults are set in `apps/api/mise.toml` dev task.

| Variable               | Default                                            | Description                     |
|------------------------|----------------------------------------------------|---------------------------------|
| `DATABASE_URL`         | `postgres://postgres:postgres@127.0.0.1:5433/api?sslmode=disable` | Postgres connection string |
| `PORT`                 | `4000`                                             | API listen port                 |
| `LOG_LEVEL`            | `info`                                             | `debug`, `info`, `warn`, `error`|
| `CORS_ALLOWED_ORIGINS` | `http://localhost:3000,...,http://localhost:5173`   | Comma-separated allowed origins. Dead weight in production: the deployed client is same-origin |
| `CLIENT_DIR`           | `./client`                                         | Where the built SPA lives. Set by the image layout, not by hand |
| `JOURNAL_URL` / `JOURNAL_TOKEN` | empty                                     | Optional log shipping to Journal |

## Conventions

- **API module pattern**: Each feature module (`modules/<name>/`) has `types.go`, `service.go`, `controller.go`, `router.go`, and `documentation.go`. Follow this pattern when adding new modules.
- **Migrations**: GORM auto-migration runs on every API startup via `schemas/migrate.go`. Add new models to `schemas/` and register them in `Migrate()`.
- **Routing**: Modules register their routes via `RegisterRoutes()` called from `main.go`, **inside `router.Route("/api", ...)`**. Anything registered at the root competes with the SPA catch-all, so only put non-API routes there (`/health`, `/ready`, `/docs`).
- **Client API calls**: always relative to `/api` (`apps/client/src/lib/backend.ts`). Never an absolute host, never a build-time `VITE_API_BASE_URL` — that is what forced the old two-container split.
- **Client package manager is bun**, with `apps/client/bun.lock` as the only lockfile. The Docker client stage runs `bun install --frozen-lockfile`, so a stale lockfile fails the image build rather than silently drifting.
- **Client UI components**: Hand-rolled Svelte 5 components in `apps/client/src/lib/components/ui/` (button, card, input, label). Uses `class-variance-authority` + `clsx` + `tailwind-merge` for style composition.
- **Svelte runes**: Runes mode is force-enabled for all non-node_modules files in `svelte.config.js`.
- **No test framework in client**: The client currently has no test runner configured.
- **Local Postgres**: `scripts/postgres-up.sh` picks a provider automatically — Docker (named volume, Postgres 16) if the daemon is up, otherwise `initdb`/`pg_ctl` from `PATH` with data in `.local/postgres/`. Runs on port 5433 (not the default 5432) to avoid conflicts. `POSTGRES_PORT`, `POSTGRES_DB`, `POSTGRES_IMAGE` and `POSTGRES_PROVIDER` override the defaults; `scripts/rename.sh` derives the port and name from the project name so forks do not collide.
- **Never put a database in `[tools]`.** `mise run` resolves every tool in the merged config before running any task body, so an uninstallable pin disables the whole task runner — `mise run check` included. The old `asdf:mise-plugins/mise-postgres@14.22` pin did exactly that: it builds from source and fails on macOS, so the quality gate could never start.

## Gotchas

- **Auth is not JWT.** `authcrypto.NewToken()` mints 32 random bytes, base64url-encoded. The token is stored verbatim in the `sessions` table with a 30-day expiry and presented as `Authorization: Bearer <token>`. There is no signing key and nothing to decode - revocation is a row delete.
- **`modules/events`, `modules/tickets`, `internal/ticketcode`, `cmd/seed`, `schemas/event.go` and `schemas/ticket.go` are leftover demo domain**, not template features. Delete them in a fork.
- The API has a `/docs` endpoint that returns auto-generated JSON documentation for all modules. It sits at the root, not under `/api`, so its URL survives the SPA catch-all.
- The API has `/health` and `/ready` endpoints for container orchestration, also at the root.
- **The SPA catch-all must be registered last.** `router.Handle("/*", ...)` swallows every path chi has not already claimed, so any route added after it is unreachable.
- **A missing hashed asset 404s instead of returning `index.html`.** `tronc/spa` does that on purpose: serving the index for a missing `.js` hides the real error behind a MIME failure.
