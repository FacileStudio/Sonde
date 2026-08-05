# GoSvelteBoilerplate — Configuration

Every variable the API actually reads, where the local defaults come from, and the knobs the
dev database and the rename script expose.

## API environment

`apps/api/internal/env/env.go` reads and validates the whole configuration at startup. Any
error it returns stops the process before the database is opened.

| Variable | Required | Default | What it does |
|---|---|---|---|
| `DATABASE_URL` | **yes** | — | Postgres connection string. Empty is a startup failure |
| `PORT` | no | `4000` | API listen port. Must parse as an integer in 1–65535 |
| `LOG_LEVEL` | no | `info` | One of `debug`, `info`, `warn`, `error`. Anything else fails |
| `CORS_ALLOWED_ORIGINS` | no | the four localhost dev origins | Comma-separated. Each entry must be `*` or start with `http://` / `https://` |
| `CLIENT_DIR` | no | `./client` | Where the built SPA lives, read by `tronc/spa` |
| `JOURNAL_URL` | no | — | Journal ingest URL. Shipping needs both this and the token |
| `JOURNAL_TOKEN` | no | — | Per-app Journal key |

The `CORS_ALLOWED_ORIGINS` default is `http://localhost:3000`, `http://127.0.0.1:3000`,
`http://localhost:5173`, `http://127.0.0.1:5173`. An explicitly empty list is rejected —
validation requires at least one origin. In the deployed shape it does not matter: the client
is served from the same origin as the API, so no browser request is ever cross-origin.

`CLIENT_DIR` is set by the image layout (`ENV CLIENT_DIR=/client` in the `Dockerfile`), not by
hand. Setting it wrong is a silent failure: the API answers, the healthcheck stays green, and
only a browser sees that the SPA is missing.

`docker-compose.yml` also passes `APP_ENV`, which this template's `internal/env` does not
read. It is there because the suite standardized on it through `troncenv.Core`; a fork that
adopts `troncenv` picks it up for free.

## Local defaults

There is no `.env` loading in the Go code. Local defaults come from the `dev` task in
`apps/api/mise.toml`, which sources `apps/api/.env` when it exists and then applies its own
fallbacks:

```sh
DATABASE_URL=postgres://postgres:postgres@127.0.0.1:5433/api?sslmode=disable
PORT=4000
LOG_LEVEL=info
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000,http://localhost:5173,http://127.0.0.1:5173
```

It prints each of them on startup, so what the process actually got is visible in the first
four lines of output. Copy `apps/api/.env.example` to `apps/api/.env` to override any of them.

## Dev database

`scripts/postgres-up.sh` is driven by four variables, all optional:

| Variable | Default | What it does |
|---|---|---|
| `POSTGRES_PORT` | `5433` in the template, rewritten per fork | Host port |
| `POSTGRES_DB` | `api` in the template, rewritten per fork | Database name |
| `POSTGRES_IMAGE` | `postgres:16-alpine` | Image used by the Docker provider |
| `POSTGRES_PROVIDER` | `auto` | `docker`, `local`, or `auto` |

`auto` picks `docker` when the daemon is reachable, otherwise `local` when `initdb`,
`pg_ctl`, `pg_isready`, `psql` and `createdb` are all on `PATH`. With neither, it says so and
names what to install. Docker is preferred because it matches the deployed Postgres 16; the
local fallback exists so a machine without a container runtime is not stuck.

| Provider | Data lives in |
|---|---|
| `docker` | Named volume `<db>_dev_pgdata`, container `<db>_dev_postgres` |
| `local` | `.local/postgres/` inside the repo |

**The port and database name are derived per fork.** `scripts/rename.sh` computes
`5433 + cksum(slug) % 200` and rewrites both, so two forks can run `mise run db-up` at the
same time without colliding. Do not hard-code 5433 in a fork's docs.

### Postgres is deliberately not in `[tools]`

`mise run` resolves *every* tool in the merged config before it runs *any* task body, so a
Postgres pin that fails to install takes the whole task runner down with it — `mise run check`
included. This template previously pinned `asdf:mise-plugins/mise-postgres@14.22`, which
builds from source and could not install on macOS, so the quality gate could never run. Do not
put a database back in `[tools]`.

## Deployment environment

`docker-compose.yml` passes, with defaults:

| Variable | Default | Notes |
|---|---|---|
| `POSTGRES_USER` | `postgres` | Used to compose `DATABASE_URL` |
| `POSTGRES_PASSWORD` | `postgres` | Change it |
| `POSTGRES_DB` | the project slug | Also names the `db_data` volume |
| `APP_ENV` | `production` | Passed through; see the note above |
| `LOG_LEVEL` | `info` | |
| `CORS_ALLOWED_ORIGINS` | empty | Same-origin in production, so it only matters for a cross-origin dev client |
| `JOURNAL_URL` / `JOURNAL_TOKEN` | empty | Optional log shipping |

`DATABASE_URL` and `PORT` are composed in the compose file itself and are not meant to be
overridden from outside.

Back to the [documentation index](README.md).
