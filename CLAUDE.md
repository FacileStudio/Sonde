# FDT (GoSvelteBoilerplate)

Monorepo boilerplate for a Go API + SvelteKit frontend, managed with mise.

## Tech Stack

- **API** (`apps/api`): Go 1.24, Chi router, GORM + Postgres, JWT auth
- **Client** (`apps/client`): SvelteKit, Svelte 5 (runes), Tailwind CSS v4, Vite 7, TypeScript
- **Tooling**: mise (monorepo mode, experimental), mise-managed Postgres 14 for local dev
- **Containerization**: Multi-stage Dockerfile (Debian slim build, distroless runtime)

## Prerequisites

- [mise](https://mise.jdx.dev/) installed and activated
- `MISE_EXPERIMENTAL=1` if monorepo tasks are not picked up automatically

## Key Commands

All commands run from the repo root unless noted.

```sh
mise trust && mise install    # one-time setup: trust config + install toolchains
mise run bootstrap            # install all app dependencies (Go modules + npm)
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
docker build -f apps/api/Dockerfile -t fdt-api .   # build from repo root
```

## Project Structure

```
.
├── apps/
│   ├── api/                  # Go API
│   │   ├── cmd/seed/         # DB seed script
│   │   ├── internal/         # Shared internal packages
│   │   │   ├── authcontext/  # Request-scoped auth context
│   │   │   ├── authcrypto/   # Password hashing + JWT tokens
│   │   │   ├── database/     # GORM database connection
│   │   │   ├── documentation/# Auto-docs type definitions
│   │   │   ├── env/          # Environment config loader
│   │   │   ├── errors/       # Structured API errors
│   │   │   ├── httpjson/     # JSON response helpers
│   │   │   ├── logger/       # slog-based logger
│   │   │   ├── middleware/    # CORS, auth, logging middleware
│   │   │   └── ticketcode/   # Ticket code generation
│   │   ├── modules/          # Feature modules (controller/service/router/types/docs per module)
│   │   │   ├── auth/
│   │   │   ├── events/
│   │   │   ├── tickets/
│   │   │   └── users/
│   │   ├── schemas/          # GORM models + auto-migration
│   │   ├── main.go           # Entrypoint
│   │   ├── mise.toml         # API-specific mise tasks
│   │   └── Dockerfile
│   └── client/               # SvelteKit frontend
│       ├── src/
│       │   ├── lib/          # Components (ui/), utils, assets
│       │   └── routes/       # SvelteKit file-based routing
│       ├── svelte.config.js
│       └── vite.config.ts
├── packages/                 # Shared packages (currently empty)
├── scripts/                  # postgres-up.sh / postgres-down.sh
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
| `CORS_ALLOWED_ORIGINS` | `http://localhost:3000,...,http://localhost:5173`   | Comma-separated allowed origins |

## Conventions

- **API module pattern**: Each feature module (`modules/<name>/`) has `types.go`, `service.go`, `controller.go`, `router.go`, and `documentation.go`. Follow this pattern when adding new modules.
- **Migrations**: GORM auto-migration runs on every API startup via `schemas/migrate.go`. Add new models to `schemas/` and register them in `Migrate()`.
- **Routing**: Modules register their routes via `RegisterRoutes()` called from `main.go`.
- **Client UI components**: Hand-rolled Svelte 5 components in `apps/client/src/lib/components/ui/` (button, card, input, label). Uses `class-variance-authority` + `clsx` + `tailwind-merge` for style composition.
- **Svelte runes**: Runes mode is force-enabled for all non-node_modules files in `svelte.config.js`.
- **No test framework in client**: The client currently has no test runner configured.
- **Local Postgres**: Managed by mise via scripts in `scripts/`. Data stored in `.local/postgres/`. Runs on port 5433 (not the default 5432) to avoid conflicts.

## Gotchas

- The `apps/api` directory has its own `.git` — it was likely initialized as a standalone repo. Be aware of this if working with git from the monorepo root.
- The `packages/` directory exists but is empty. It is reserved for future shared packages.
- The API has a `/docs` endpoint that returns auto-generated JSON documentation for all modules.
- The API has `/health` and `/ready` endpoints for container orchestration.
