# API

Small Go backend for authentication, events, and ticket check-in.

Built with:
- Chi for routing
- GORM + Postgres for persistence
- JWT auth for protected endpoints

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

Build the production image from the repo root so Docker can access the shared `mise` config and lockfile:

```bash
docker build -f apps/api/Dockerfile -t fdt-api .
```

## API

Auth:

```bash
curl -sS -X POST http://localhost:4000/auth/register \
  -H 'content-type: application/json' \
  -d '{"email":"me@example.com","password":"password123"}'

curl -sS -X POST http://localhost:4000/auth/login \
  -H 'content-type: application/json' \
  -d '{"email":"me@example.com","password":"password123"}'

curl -sS http://localhost:4000/users/me \
  -H "Authorization: Bearer $TOKEN"

curl -sS -X PATCH http://localhost:4000/users/me \
  -H 'content-type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"email":"updated@example.com","password":"newpassword123"}'
```

Events:

```bash
curl -sS http://localhost:4000/events

curl -sS http://localhost:4000/events/$EVENT_ID

curl -sS -X POST http://localhost:4000/events \
  -H 'content-type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"My Event","starts":"2030-01-01T18:00:00Z","ends":"2030-01-01T21:00:00Z"}'
```

Tickets:

```bash
curl -sS -X POST http://localhost:4000/events/$EVENT_ID/tickets \
  -H "Authorization: Bearer $TOKEN"

curl -sS -X POST http://localhost:4000/tickets/validate \
  -H 'content-type: application/json' \
  -d '{"code":"'"$CODE"'"}'

curl -sS -X POST http://localhost:4000/tickets/checkin \
  -H 'content-type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"code":"'"$CODE"'"}'
```
