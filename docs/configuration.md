# Sonde — Configuration

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
| `SONDE_RAW_RETENTION_DAYS` | no | `30` | Days of raw checks kept before the pruner deletes them |
| `SONDE_FLEET_TOKEN` | no | empty | Bearer token the fleet runner presents to `POST /api/fleet/reports`. Empty turns ingest off (404). At least 32 characters, or the app refuses to boot |
| `SONDE_FLEET_ADMINS` | no | empty | Comma-separated emails allowed to read fleet data. The account must also be an administrator. Empty closes every fleet read route |
| `SONDE_FLEET_DISPATCH_TOKEN` | no | empty | Fine-grained GitHub token, Actions read and write on FacileStudio/Sonde only. Lets the Maintenance page queue a rescan. Empty answers 412 |

The `CORS_ALLOWED_ORIGINS` default is `http://localhost:3000`, `http://127.0.0.1:3000`,
`http://localhost:5173`, `http://127.0.0.1:5173`. An explicitly empty list is rejected —
validation requires at least one origin. In the deployed shape it does not matter: the client
is served from the same origin as the API, so no browser request is ever cross-origin.

`CLIENT_DIR` is set by the image layout (`ENV CLIENT_DIR=/client` in the `Dockerfile`), not by
hand. Setting it wrong is a silent failure: the API answers, the healthcheck stays green, and
only a browser sees that the SPA is missing.

`docker-compose.yml` also passes `APP_ENV`, which `internal/env` does not read. It is there
because the suite standardized on it through `troncenv.Core`; adopting `troncenv` picks it up
for free.

`SONDE_RAW_RETENTION_DAYS` fails soft, unlike everything above it. `intOrDefault` returns 30
for a value that is not an integer or is below 1, so a typo boots on the default instead of
stopping the process. [docs/RETENTION.md](RETENTION.md) covers the three tiers and what each
one keeps.

## Authentication

Sonde authenticates through porte. `OIDC_ISSUER` is the switch: set it and single sign-on
turns on, which makes the rest of the OIDC set required. `env.Load` refuses a half-filled set
at startup rather than failing on somebody's first login attempt.

| Variable | Required | Default | What it does |
|---|---|---|---|
| `APP_ENV` | no | — | `production` refuses to boot without `OIDC_ISSUER`. The first account ever created is admin |
| `OIDC_ISSUER` | no | — | `https://sso.facile.studio`. Empty disables SSO |
| `OIDC_CLIENT_ID` | with issuer | — | `sonde` |
| `OIDC_CLIENT_SECRET` | with issuer | — | Set on the Dokploy service, never committed |
| `OIDC_REDIRECT_URL` | with issuer | — | `https://sonde.facile.studio/api/auth/oidc/callback` |
| `OIDC_SUCCESS_URL` | with issuer | — | Where a completed login lands. Its origin also supplies the default failure URL |
| `OIDC_CLI_AUDIENCE` | no | empty | Mounts porte's device-exchange route so the companion CLI can sign in. Production value is `facile-cli`. Empty leaves the route unmounted |

**The identity provider is Registre at `https://sso.facile.studio`, and it serves one issuer
at the root.** Authentik is gone, `porte.facile.studio` 404s, and the old per-app issuer path
`https://porte.facile.studio/application/o/sonde/` resolves to nothing. go-oidc compares the
discovery document's `issuer` byte for byte, so there is no alias or trailing-slash variant
that papers over a stale value.

**`OIDC_REDIRECT_URL` must keep the `/api` prefix.** porte mounts under `/api`, and a bare
path does not 404: it lands on the SvelteKit catch-all and returns 200 with HTML, so the
login silently never completes.

`TRUST_EMAIL_WITHOUT_VERIFIED_CLAIM` is not offered. Registre reports `email_verified`
truthfully, so the flag would buy nothing. [docs/AUTH.md](AUTH.md) has the flow and the
failure-redirect contract.

## Event publishing

Sonde publishes events to Antenne through antenne-client's outbox. The checker queues each
outage and recovery in `antenne_outbox`, in the same transaction as the incident write, and a
drainer sends the queue in order once the bus answers. An event raised while Antenne is down
waits in the table instead of being dropped; sent rows are pruned after 7 days.

Publishing is off unless `ANTENNE_URL` or its older spelling `ANTENNE_INSTANCE` is set. With
neither, nothing dials, nothing is queued, and the checker runs exactly as it does without it.

| Variable | Required | Default | What it does |
|---|---|---|---|
| `ANTENNE_URL` | no | — | The Antenne instance to publish to. Wins over `ANTENNE_INSTANCE` |
| `ANTENNE_INSTANCE` | no | — | Older spelling of `ANTENNE_URL`, still read. Both unset disables publishing entirely |
| `ANTENNE_APP` | no | `sonde` | The app name events are attributed to |
| `ANTENNE_SECRET` | with a URL | — | Shared secret presented when registering |
| `ANTENNE_INSTANCE_ID` | no | — | Optional. Distinguishes two deployments of the same app |

## Local defaults

There is no `.env` loading in the Go code. Local defaults come from the `dev` task in
`apps/api/mise.toml`, which sources `apps/api/.env` when it exists and then applies its own
fallbacks:

```sh
DATABASE_URL=postgres://postgres:postgres@127.0.0.1:5603/sonde?sslmode=disable
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
| `APP_VERSION` | `dev` | Build arg served at `GET /version`; set it in the Dokploy env at release |
| `OIDC_ISSUER` / `OIDC_CLIENT_ID` / `OIDC_CLIENT_SECRET` / `OIDC_REDIRECT_URL` / `OIDC_SUCCESS_URL` | empty | All or nothing; see Authentication above |
| `OIDC_CLI_AUDIENCE` | empty | `facile-cli` in production |
| `ANTENNE_URL` / `ANTENNE_INSTANCE` / `ANTENNE_SECRET` / `ANTENNE_INSTANCE_ID` | empty | Both URL spellings empty disables publishing |
| `ANTENNE_APP` | `sonde` | |
| `SONDE_RAW_RETENTION_DAYS` | `30` | |

`DATABASE_URL` and `PORT` are composed in the compose file itself and are not meant to be
overridden from outside. Every secret above is set on the Dokploy service, not committed.

Back to the [documentation index](README.md).
