# GoSvelteBoilerplate — Deployment

The one-container shape every Facile Go service inherits, why it is shaped that way, and how
to check a deploy actually works.

## The rule

**One container, one Traefik router, one hostname.**

```
<app>.facile.studio         ──▶ the Go binary
<app>.facile.studio/api/*   ──▶ the API routes, under router.Route("/api", …)
everything else             ──▶ the built SvelteKit SPA, served by tronc/spa
```

There is no client container. There is no second router. `docker-compose.yml` carries the
canonical explanation in its header comment — read it before changing anything about routing,
and keep it when you fork.

## The image

The root `Dockerfile` is the only Dockerfile in the repo. There is no `apps/client/Dockerfile`
and no `apps/api/Dockerfile`. Three stages:

1. `oven/bun:1` copies `apps/client/package.json` and `bun.lock` first, runs
   `bun install --frozen-lockfile`, then copies the source and builds. Lockfile first means a
   source change does not invalidate the install layer; `--frozen-lockfile` means a stale
   lockfile fails the build instead of drifting.
2. `golang:1.24-alpine` does the same trick with `go.mod` / `go.sum`, then builds with
   `CGO_ENABLED=0` and `-trimpath -ldflags="-s -w"`.
3. `gcr.io/distroless/static-debian12:nonroot` receives the binary at `/api` and the built
   client at `/client`, sets `CLIENT_DIR=/client`, exposes `4000`, and runs as
   `nonroot:nonroot`.

Two things in that last stage are load-bearing:

- **`CLIENT_DIR` is set explicitly.** The distroless base can carry its own `WorkingDir`
  (`/home/nonroot` on the `:nonroot` variant), which would make a relative `./client` resolve
  there and the SPA silently not be served at all.
- **`:nonroot` is safe only because the container writes nothing.** A fork that mounts a
  writable volume **must drop `:nonroot` and the `USER` line** — a named volume is created
  owned by root and a non-root process cannot write to it.

## Compose topology

```
Traefik ──▶ <slug>-svc :4000 ──▶ /api binary ──▶ db :5432 (postgres:16-alpine)
                                                     │
                                              db_data volume
```

The `api` service is `expose`d and never `ports`-published, so only Traefik reaches it over
the shared external `dokploy-network`. Two routers point at the same service: `<slug>-web` on
the `web` entrypoint redirecting through `redirect-to-https@file`, and `<slug>-secure` on
`websecure` with the `letsencrypt` cert resolver. Both use `Host(<slug>.facile.studio)`.

**No `PathPrefix`, no `stripprefix`.** A single router takes the whole host and the Go router
decides what is API and what is SPA. The routes live under `/api` in `main.go`, so the prefix
must reach the binary intact — a `stripprefix` middleware would 404 every route.

Compose labels **replace** what Dokploy injects for a service; they do not merge with it.
Declaring them here therefore takes full ownership of this service's routing, which is the
point.

If a fork needs a route that is neither `/api` nor the SPA — Nuage serves `/webdav` that way —
add another router with its own rule pointing at the **same service**. Do not add a second
service.

The `db` service waits on `pg_isready` and the API `depends_on` it with
`condition: service_healthy`, so a cold start does not race the first migration.

## Healthchecks

```yaml
test: ["CMD", "/api", "healthcheck"]
```

The image is distroless: no shell, no `wget`, no `curl`. Re-executing the binary is the only
probe available, and `tronc/healthcheck` answers it before any configuration is read, so a
misconfigured process still fails on the right thing.

Over HTTP, `tronc/health` mounts `/health` (liveness, touches nothing) and `/ready`
(readiness, pings the database) at both the root and under `/api` — so the probe works whether
or not the edge proxies only `/api/*`.

## Deploying

Deploys go through Dokploy on la ruche, panel at
[gare.facile.studio](https://gare.facile.studio), with Traefik in front. Prefer the `dokploy`
CLI over SSH and docker. Environment variables are set on the Dokploy service, not committed —
see [configuration.md](configuration.md).

Locally:

```sh
docker compose up -d --build
docker compose logs -f api
```

## Migrations

GORM auto-migration runs on every startup, from `schemas.Migrate`. There is no separate
migration step and no migration tool: deploying the new binary migrates the database. Add a
model to `schemas/` and register it in `Migrate()`.

That has a consequence worth planning around: a destructive schema change is not expressible
this way. `AutoMigrate` adds columns and indexes; it does not drop or rewrite them. Anything
beyond additive goes in as raw SQL in `Migrate`, written to be idempotent — the
`tickets_status_check` constraint is the worked example, wrapped in
`DO $$ … EXCEPTION WHEN duplicate_object THEN NULL`.

## Verifying a deploy

**A green `/api/health` says nothing about what the browser calls.** That is exactly the blind
spot that let a broken frontend reach production once. `scripts/verify-deploy.sh` checks both
halves:

```sh
sh scripts/verify-deploy.sh myproject
sh scripts/verify-deploy.sh myproject /api/auth/config
```

Nine checks, in order: `/` returns 200 with `text/html`; an unknown client-side route falls
back to the index; a hashed asset actually referenced by the page resolves; a missing bundle
404s rather than receiving `index.html`; the boot API path answers 200 with
`application/json`; the same path with the prefix doubled (`/api/api/…`) 404s; `/api/health`
and `/api/ready` are 200; a `POST` to an unknown route is refused with 405 rather than
absorbed as a 2xx; and no bundle the page actually loads contains `/api/api`. The second
argument overrides the boot path, the third the domain. Non-zero exit means the deploy is
broken however green the container looks.

Back to the [documentation index](README.md).
