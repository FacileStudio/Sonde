# Sonde

Self-hosted uptime monitoring for the Facile Suite. Sonde probes your services, records every
result, opens and resolves incidents, and publishes status pages. A Go API and a SvelteKit
client deploy as one container behind one Traefik router on one hostname.

It is the suite-native answer to Uptime Kuma. Auth comes from porte, logs go to Journal,
alerts go to Antenne, and the check history prunes itself instead of growing forever.

## What it does

- Monitors **HTTP** endpoints (status code and keyword), **TCP** ports, and **push**
  heartbeats for things that run somewhere Sonde cannot reach
- Opens an incident when a monitor goes down and resolves it when the monitor comes back
- Reports uptime over 24 hours, 7 days, 30 days and 90 days, from hourly rollups
- Serves public status pages at `/status/{slug}`, no login required
- Exports monitors and status pages as JSON, secrets excluded
- Rolls up raw checks into hourly and daily buckets and prunes what expires, on a daily job
- Signs in with argon2id passwords or Registre SSO, through one session
- Ships both apps as a single distroless image, no client container

## Stack

| Layer | Tech |
|---|---|
| API | Go 1.25, Chi v5, GORM + PostgreSQL 16, [tronc](https://github.com/FacileStudio/tronc) 0.12.0 |
| Auth | [porte](https://github.com/FacileStudio/porte) 0.3.1, argon2id and OIDC through one session manager |
| Client | SvelteKit 2 (`adapter-static`), Svelte 5 (runes forced on), Tailwind CSS 4, Vite 7, [`@facile/muse`](https://github.com/FacileStudio/muse) |
| Deploy | One multi-stage Dockerfile, Docker Compose, distroless `:nonroot` behind Traefik |
| Runtime | mise (monorepo mode), bun 1.3.11 for the client |

`tronc` supplies `errors`, `httpjson`, `logger`, `middleware`, `httpx`, `health`,
`healthcheck`, `spa`, `apiref`, `migrate` and `testdb`. Do not reimplement any of them
locally.

## Quick start

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
mise run check          # the quality gate: gofmt, go vet, go test, svelte-check
mise run build          # build both apps
docker build -t sonde .
```

`scripts/check.sh` is the gate, and the tracked `pre-push` hook runs it. It depends on nothing
but a `go` on `PATH` and `bun` for the client, and it is deliberately not invoked through
mise: `mise run` resolves every tool in the merged config before running any task body, so an
unrelated broken tool in a global config would take the gate down with it.

The Go half also fails on a non-Postgres GORM driver in `go.mod`. Postgres is the only
database here, tests included. A test on SQLite builds a different schema from the struct tags
and then passes, proving nothing about the DDL that ships. `tronc/testdb` gives a test a real
Postgres; set `TEST_DATABASE_URL` to enable those tests, and they skip cleanly without it.

[filet](https://github.com/FacileStudio/filet) checks style, complexity and layout against
`filet.yml`, inherited verbatim from the Go family's fork source. Its limits are the published
defaults of the tools they come from rather than numbers picked to make a count go away.

```sh
filet check .    # the rules
filet roast .    # the same rules, with commentary
```

What fails is `error`, which here means the `architecture` block: a package under
`apps/api/modules/` that owns HTTP routes and has no `router.go`, or a missing top-level
directory. Style and complexity findings are warnings, worth reading and not a merge blocker.
`.github/workflows/filet.yml` runs the same gate on push and on pull requests, and reports to
Antenne through `ANTENNE_FILET_SECRET`. Without that secret the workflow withholds the URL
rather than failing, so a pull request from a fork does not break on a secret it cannot have.

## Configuration

The API is configured entirely through the environment. Local defaults live in the `apps/api`
dev task; copy `apps/api/.env.example` to `apps/api/.env` for overrides.

| Variable | What it does |
|---|---|
| `DATABASE_URL` | Postgres connection string. The only required variable |
| `PORT` | API listen port, default `4000` |
| `LOG_LEVEL` | `debug`, `info`, `warn` or `error` |
| `SONDE_RAW_RETENTION_DAYS` | Days of raw checks kept, default 30 |
| `OIDC_ISSUER` and friends | Registre SSO. `https://sso.facile.studio` |
| `ANTENNE_INSTANCE` and friends | Event publishing. Empty disables it |
| `CLIENT_DIR` | Where the built SPA lives. Set by the image layout, not by hand |

Full reference: [docs/configuration.md](docs/configuration.md).

## Retention

| Table | Kept for |
|---|---|
| `checks_raw` | 30 days, tunable through `SONDE_RAW_RETENTION_DAYS` |
| `checks_hourly` | 13 months |
| `checks_daily` | Forever |

A job inside the API process rolls the buckets up and prunes what expired, every 24 hours.
The rollups hold a monitor id, a timestamp and three counters, and nothing else, which is what
makes keeping them indefinitely storage-limitation clean.
[docs/RETENTION.md](docs/RETENTION.md) has the reasoning.

## Structure

```
apps/
  api/       Go API. internal/ (the domain), modules/ (auth, users), migrations/, schemas/
  client/    SvelteKit frontend, adapter-static, built into apps/client/build
scripts/     rename.sh, check.sh, postgres-up.sh, postgres-down.sh, verify-deploy.sh
docs/        Architecture, auth, retention, configuration, development, deployment, API
Dockerfile   The only Dockerfile: builds both apps into one image
```

`docker-compose.yml` is the canonical deployment shape. Read its header comment before
changing anything about routing.

## Documentation

| Doc | What's in it |
|---|---|
| [Architecture](docs/architecture.md) | Request flow, where the code lives, data model, auth |
| [Authentication](docs/AUTH.md) | porte, Registre SSO, CSRF, the failure-redirect contract |
| [Retention](docs/RETENTION.md) | The three data tiers, the prune job, the RGPD reasoning |
| [Configuration](docs/configuration.md) | Every environment variable and default |
| [Development](docs/development.md) | The dev database, the quality gate, the rename script |
| [Deployment](docs/deployment.md) | The one-container shape, Traefik routing, verifying a deploy |
| [API](docs/api.md) | HTTP endpoints and payloads |

---

Part of the [Facile Suite](https://facile.studio), self-hosted tools for creative studios
and freelancers. One login, zero cloud dependency.
