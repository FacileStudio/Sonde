# API

Small Go backend: a Chi router, GORM models, and a session-based auth module.

Built with:
- Chi for routing
- GORM + Postgres for persistence
- Opaque bearer tokens for protected endpoints - `authcrypto.NewToken()` mints 32 random
  bytes, stored in the `sessions` table with a 30-day expiry. Not JWT: nothing is signed
  and nothing is decodable, so revoking a session is a row delete.

The `events` and `tickets` modules (plus `internal/ticketcode`, `cmd/seed`, and the matching
schemas) are leftover demo domain, not features of the template. Delete them in a fork.

## Quick Start

Preferred requirements:
- `mise`
- Postgres

Run from the repo root:

```bash
mise install
cp apps/api/.env.example apps/api/.env
mise //apps/api:dev
```

If you prefer to work inside `apps/api`, the local task form works too:

```bash
mise :dev
```

The server listens on `:4000` by default. Set `PORT` to override it.
By default CORS allows local frontend origins on ports `3000` and `5173`. Override with `CORS_ALLOWED_ORIGINS` as a comma-separated list.
Set `LOG_LEVEL` to one of `debug`, `info`, `warn`, or `error`.

On startup the app runs GORM auto-migrations for all tables.

## Common Commands

```bash
mise //apps/api:dev        # run the API
mise //apps/api:test       # run tests
mise //apps/api:check      # fmt + vet + test
mise //apps/api:build      # build ./bin/api
```

`mise` manages the Go toolchain for this repo, and the API command flow now lives directly in `apps/api/mise.toml`.

## Docker

There is no `apps/api/Dockerfile`. The repo ships **one** image, built from the root
`Dockerfile`, holding both the API binary and the built SvelteKit client:

```bash
docker build -t gosvelteboilerplate .
```

The binary serves `/api/*` from its own routes and everything else from the static client in
`CLIENT_DIR` (default `./client`). See "Deployment shape" in the root `CLAUDE.md`.

## API

Every application route lives under `/api`; `/health`, `/ready` and `/docs` stay at the root
so the SPA catch-all cannot shadow them.

Auth:

```bash
curl -sS -X POST http://localhost:4000/api/auth/register \
  -H 'content-type: application/json' \
  -d '{"email":"me@example.com","password":"password123"}'

curl -sS -X POST http://localhost:4000/api/auth/login \
  -H 'content-type: application/json' \
  -d '{"email":"me@example.com","password":"password123"}'

curl -sS http://localhost:4000/api/users/me \
  -H "Authorization: Bearer $TOKEN"

curl -sS -X PATCH http://localhost:4000/api/users/me \
  -H 'content-type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"email":"updated@example.com","password":"newpassword123"}'
```

Events (demo domain):

```bash
curl -sS http://localhost:4000/api/events

curl -sS http://localhost:4000/api/events/$EVENT_ID

curl -sS -X POST http://localhost:4000/api/events \
  -H 'content-type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"My Event","starts":"2030-01-01T18:00:00Z","ends":"2030-01-01T21:00:00Z"}'
```

Tickets (demo domain):

```bash
curl -sS -X POST http://localhost:4000/api/events/$EVENT_ID/tickets \
  -H "Authorization: Bearer $TOKEN"

curl -sS -X POST http://localhost:4000/api/tickets/validate \
  -H 'content-type: application/json' \
  -d '{"code":"'"$CODE"'"}'

curl -sS -X POST http://localhost:4000/api/tickets/checkin \
  -H 'content-type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"code":"'"$CODE"'"}'
```
