# Sonde

The template every Facile Go service is forked from: a Go API and a SvelteKit client that
deploy as one container behind one Traefik router on one hostname.

Fork it, run `scripts/rename.sh`, delete the demo domain, and you have a working app with the
suite's deployment shape, quality gate and chassis already wired. What is in here is not a
suggestion — a fork that deviates from the routing shape breaks in production, which is why
`docker-compose.yml` carries a header comment longer than the file.

## What it does

- Ships a Go API and a SvelteKit client as a single distroless image, no client container
- Routes `/api/*` to the API and everything else to the SPA, from one Go router
- Provides auth out of the box: argon2id passwords and DB-backed session tokens
- Applies ordered SQL migrations on startup via `tronc/migrate` and serves generated module docs at `/docs`
- Runs a repo-local dev Postgres through Docker, falling back to `PATH` binaries
- Enforces a quality gate — gofmt, vet, test, svelte-check, filet — from a pre-push hook and from CI
- Renames itself into a new project in one pass: module path, imports, docs, dev database

## Stack

| Layer | Tech |
|---|---|
| API | Go 1.25, Chi v5, GORM + PostgreSQL 16, [tronc](https://github.com/FacileStudio/tronc) 0.9.0 |
| Client | SvelteKit 2 (`adapter-static`), Svelte 5 (runes forced on), Tailwind CSS 4, Vite 7 |
| Deploy | One multi-stage Dockerfile, Docker Compose, distroless `:nonroot` behind Traefik |
| Runtime | mise (monorepo mode), bun 1.3.11 for the client |

`tronc` supplies `env`, `errors`, `httpjson`, `logger`, `middleware`, `httpx`, `health`,
`healthcheck` and `spa`. Do not reimplement any of them locally.

## Quick start

Forking is the first thing you do, before anything else:

```sh
git clone git@github.com:FacileStudio/Sonde.git MyProject
cd MyProject
scripts/rename.sh MyProject
```

Then get it running:

```sh
mise trust && mise install
mise run bootstrap
mise run hooks
mise run db-up
mise run dev
```

The API listens on `:4000`, the client dev server on `:5173`, and Vite proxies `/api` to the
API so local and production take the same relative path.

```sh
mise run check          # the quality gate: gofmt, vet, test, svelte-check, filet
mise run build          # build both apps
docker build -t myproject .
```

## The style gate

[filet](https://github.com/FacileStudio/filet) checks style, complexity and layout against
`filet.yml` at the repository root. Its limits are the published defaults of the tools they
come from — funlen, SonarSource cognitive complexity, ESLint max-lines and max-depth, lll —
rather than numbers picked to make a count go away, and every exception in that file names
its reason.

```sh
GOBIN=$HOME/go/bin go install github.com/FacileStudio/filet@v0.3.0   # once
filet check .    # the rules
filet roast .    # the same rules, with commentary
filet docker .   # the Dockerfile
```

Install it with an explicit `GOBIN`: mise exports a per-Go-version `GOBIN`, so a plain
`go install` drops the binary somewhere that leaves `PATH` when this repo's Go version does.
`scripts/check.sh` runs filet when it finds one and skips it with a note when it does not, so
the gate still works on a machine that has only `go` and `bun`. **CI is the authority** —
`.github/workflows/filet.yml` pins the version.

What the gate actually fails on is `error`, which here means the `architecture` block in
`filet.yml`: a module under `apps/api/modules/` that owns HTTP routes and has no `router.go`,
or a missing top-level directory. Style and complexity findings are warnings — they are worth
reading, they do not stop a merge. Raising `fail-on` to `warn` is a per-fork decision, and one
worth taking only after the warnings are at zero.

### Reporting to Antenne

The workflow posts every run to Antenne, which
delivers a failure on the default branch and stays quiet otherwise — a green run says nothing,
and a failure on a pull request branch is already an annotation on the author's own diff. Set
the `ANTENNE_FILET_SECRET` repository secret; without it the workflow withholds the URL, so a
pull request from a fork does not fail on a secret it cannot have. A delivery that fails is a
warning annotation, never a red build: the alerting service being down says nothing about
this code.

## Configuration

The API is configured entirely through the environment. Defaults for local development live
in the `apps/api` dev task; copy `apps/api/.env.example` to `apps/api/.env` for overrides.

| Variable | What it does |
|---|---|
| `DATABASE_URL` | Postgres connection string. The only required variable |
| `PORT` | API listen port, default `4000` |
| `LOG_LEVEL` | `debug`, `info`, `warn` or `error` |
| `CORS_ALLOWED_ORIGINS` | Comma-separated origins. Dead weight in production — the client is same-origin |
| `CLIENT_DIR` | Where the built SPA lives. Set by the image layout, not by hand |

Full reference: [docs/configuration.md](docs/configuration.md).

## Structure

```
apps/
  api/       Go API — modules/ (auth, users, and the demo domain), schemas/ (GORM), internal/
  client/    SvelteKit frontend, adapter-static, built into apps/client/build
scripts/     rename.sh, check.sh, postgres-up.sh, postgres-down.sh, verify-deploy.sh
filet.yml    The style, complexity and architecture rules the gate enforces
docs/        Architecture, configuration, development, deployment, API
Dockerfile   The only Dockerfile: builds both apps into one image
```

`docker-compose.yml` is the canonical deployment shape. Read its header comment before
changing anything about routing.

## Documentation

| Doc | What's in it |
|---|---|
| [Architecture](docs/architecture.md) | Request flow, the module pattern, data model, auth |
| [Configuration](docs/configuration.md) | Every environment variable and default |
| [Development](docs/development.md) | Forking, the rename script, the dev database, the quality gate |
| [Deployment](docs/deployment.md) | The one-container shape, Traefik routing, verifying a deploy |
| [API](docs/api.md) | HTTP endpoints and payloads |

---

Part of the [Facile Suite](https://facile.studio) — self-hosted tools for creative studios
and freelancers. One login, zero cloud dependency.
