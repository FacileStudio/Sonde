# GoSvelteBoilerplate — Development

Forking the template, running it locally, and the quality gate every fork inherits.

## Prerequisites

| Tool | Version | Why |
|---|---|---|
| mise | any | Task runner and toolchain pins. Activate it in your shell |
| Go | 1.24 | Pinned in `mise.toml`; `apps/api/go.mod` declares `go 1.24.0` |
| bun | 1.3.11 | Pinned in `mise.toml`. The client's only package manager |
| Docker | any | Preferred dev database provider. Optional — see the fallback below |

If monorepo tasks are not picked up automatically, `export MISE_EXPERIMENTAL=1` before running
anything.

## Forking

This is the first thing you do with a clone, before writing a line of code.

```sh
git clone git@github.com:FacileStudio/GoSvelteBoilerplate.git MyProject
cd MyProject
scripts/rename.sh MyProject
```

### Why the rename is not optional

A fresh fork inherits the Go module path
`github.com/FacileStudio/GoSvelteBoilerplate/apps/api`. Leaving it — or replacing it with a
bare `module api`, which is the other thing people did — means nothing in the suite can import
your packages, and no shared Go module can ever be introduced without touching every fork
first. Seven forks shipped that way before it was caught and fixed one repo at a time. The
script exists so that never has to happen again.

### What `scripts/rename.sh MyProject` does, in one pass

- Rewrites `module` in `apps/api/go.mod` to `github.com/FacileStudio/MyProject/apps/api`
- Rewrites every scoped Go import of the old module path, package root by package root
- Deletes the `<!-- fork:begin -->` … `<!-- fork:end -->` blocks and replaces the template name
  and slug in `README.md`, `CLAUDE.md`, `apps/api/README.md`, `apps/client/README.md` and
  `docker-compose.yml` — the last one matters because it carries the public hostname and the
  Traefik router names, and a fork that skips it deploys itself onto the template's domain
- Derives a dev database name and port from your project name (`5433 + cksum(slug) % 200`) and
  rewrites `scripts/postgres-up.sh`, `scripts/postgres-down.sh`, `apps/api/mise.toml` and the
  `.env` files, so two forks can run `mise run db-up` side by side
- Rewrites `tokenStorageKey` in `apps/client/src/lib/backend.ts` to `<slug>.token`

It refuses to run on a dirty tree, so the rename stays reviewable as a single commit. It is
safe to re-run: it reports and exits if the fork has already been renamed. `MODULE_OWNER`
overrides the `FacileStudio` org.

### Then, by hand

```sh
git remote set-url origin git@github.com:FacileStudio/MyProject.git
mise //apps/api:tidy
cp apps/api/.env.example apps/api/.env
```

Delete the demo domain: `modules/events`, `modules/tickets`, `internal/ticketcode`,
`cmd/seed`, `schemas/event.go`, `schemas/ticket.go`, their registrations in `main.go`, and the
matching client code. Sweep `docs/` too — the rename script does not touch this folder.

## Running it

```sh
mise trust && mise install     # trust the config, install the toolchains
mise run bootstrap             # go mod download + bun install
mise run hooks                 # enable the pre-push quality gate in this clone
mise run db-up                 # start the repo-local Postgres
mise run dev                   # API on :4000, client on :5173
```

`mise run dev` runs both dev servers. Vite proxies `/api` to `http://127.0.0.1:4000`, so the
client takes the same relative path locally that it takes in production.

### Task reference

```sh
mise tasks ls --all            # every repo and app task
mise run build                 # build both apps
mise run db-down               # stop the dev database
mise //apps/api:dev            # API only
mise //apps/api:test           # go test ./...
mise //apps/api:build          # binary at apps/api/bin/api
mise //apps/api:tidy           # go mod tidy
mise //apps/client:dev         # Vite only
mise //apps/client:build       # production client build
mise //apps/client:check       # svelte-check
docker build -t myproject .    # one image, from the repo root
```

The app-level tasks call `go` and `bun` through `$GO_BIN` and `$BUN_BIN`, resolved from the
mise-installed toolchain, so they do not depend on whatever happens to be first on `PATH`.

## The quality gate

`scripts/check.sh` is the gate. It reports and never rewrites, except with `--format`.

```sh
sh scripts/check.sh             # gofmt -l, go vet, go test on apps/api, then the client
sh scripts/check.sh --go-only   # Go only
sh scripts/check.sh --format    # rewrite Go sources in place
```

Entry points in `mise.toml`:

| Task | Runs |
|---|---|
| `mise run check` | `sh ./scripts/check.sh` |
| `mise run check-go` | `sh ./scripts/check.sh --go-only` |
| `mise run format` | `sh ./scripts/check.sh --format` |
| `mise run hooks` | `git config core.hooksPath .githooks` |
| `mise //apps/api:check` | `sh ../../scripts/check.sh --go-only` |

### The pre-push hook

`.githooks/pre-push` runs the gate before every push. Git does not use tracked hooks by
default, so it has to be enabled once per clone:

```sh
git config core.hooksPath .githooks     # or: mise run hooks
```

**The hook calls `scripts/check.sh` directly, not through mise, and that is deliberate.**
`mise run` resolves every tool in the merged config — including your global one — before it
runs any task body. One unrelated broken tool pin anywhere in that merge would take the gate
down with it, and a gate that cannot start is a gate that is not enforcing anything. The
script depends on nothing but a `go` on `PATH` and `bun` for the client.

### The `GOROOT` trap

`check.sh` resolves the toolchain from `GOROOT` when it is set. mise exports `GOROOT` for the
version this repo pins but leaves an unrelated `go` earlier on `PATH` — Homebrew's, typically
— and a `go` binary driving a different `GOROOT` fails with
`compile: version "X" does not match go tool version "Y"`. Override with `GO` and `GOFMT` if
you need something else.

The client half is skipped with a message when `bun` is not on `PATH`, so the gate still runs
usefully in a Go-only environment.

### Bypassing

```sh
git push --no-verify
```

Only for a failure genuinely unrelated to your change.

## No CI workflows

**This suite does not add GitHub Actions workflows to its apps.** There is no `.github/`
directory here and a fork should not grow one. The gate runs on the machine that is about to
push, where it is fast and where the fix is one command away; a workflow would duplicate it,
lag it by minutes, and add a second thing to keep green across thirty-five repos. Deploys are
handled by Dokploy watching the branch, not by a pipeline.

The one exception in the suite is a tag-triggered release workflow for a repo that ships a
binary through GoReleaser. That is a distribution job, not a check.

## Conventions

- **Module pattern**: `types.go`, `service.go`, `controller.go`, `router.go`,
  `documentation.go` per module. See [architecture.md](architecture.md).
- **Migrations**: GORM auto-migration on every startup, from `schemas/migrate.go`. Add the
  model to `schemas/` and register it in `Migrate()`.
- **Routing**: modules register inside `router.Route("/api", …)`. Only `/health`, `/ready` and
  `/docs` belong at the root; anything else there competes with the SPA catch-all.
- **Client API calls** are always relative to `/api`. Never an absolute host, never a
  build-time `VITE_API_BASE_URL`.
- **bun is the client's only package manager**, with `apps/client/bun.lock` as the only
  lockfile. The Docker client stage runs `bun install --frozen-lockfile`, so a stale lockfile
  fails the image build rather than silently drifting.
- **No inline comments in code.** A comment explains *why*, above the declaration.

Back to the [documentation index](README.md).
