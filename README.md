# GoSvelteBoilerplate

GoSvelteBoilerplate is a small monorepo with:

- `apps/api`: Go API
- `apps/client`: SvelteKit frontend

The repo is managed with [`mise`](https://mise.jdx.dev/) so tool versions and common tasks are shared from the repository itself.
That includes a `mise`-managed Postgres toolchain for the local dev database.

## Fork This Template

Run the rename script first. It is not optional: a fresh fork inherits the Go module path
`github.com/FacileStudio/GoSvelteBoilerplate/apps/api`, which is wrong for your project and
breaks any attempt to import your packages from elsewhere in the suite.

```sh
git clone git@github.com:FacileStudio/GoSvelteBoilerplate.git MyProject
cd MyProject
scripts/rename.sh MyProject
```

The script requires a clean git tree and rewrites, in one pass:

- the module path in `apps/api/go.mod` to `github.com/FacileStudio/MyProject/apps/api`
- every scoped Go import of the old module path
- the template name across the README files and `CLAUDE.md`
- the dev database name and port, derived from your project name, so two forks can run
  `mise run db-up` at the same time without colliding

Then configure the rest by hand:

- `git remote set-url origin` to your own repository
- `mise //apps/api:tidy` to resettle `go.sum`
- `apps/api/.env` (copied from `.env.example`) for local overrides
- delete the demo domain: `modules/events`, `modules/tickets`, `internal/ticketcode`,
  `cmd/seed`, `schemas/event.go`, `schemas/ticket.go`, and their registrations in `main.go`

## Quick Start

1. Install and activate `mise` for your shell.
2. Trust this repository config:

   ```sh
   mise trust
   ```

3. Install the repo toolchains:

   ```sh
   mise install
   ```

4. Start the repo-local dev Postgres database:

   ```sh
   mise run db-up
   ```

5. Start both apps:

   ```sh
   mise run dev
   ```

The repo Postgres task starts a local cluster on `127.0.0.1:5433`. Override the port and
database name with `POSTGRES_PORT` and `POSTGRES_DB`. Copy `apps/api/.env.example` to
`apps/api/.env` if you want local overrides.

## Common Commands

```sh
mise tasks ls --all          # show all repo and app tasks
mise run bootstrap           # install app dependencies
mise run db-up              # start the repo-local Postgres database on 127.0.0.1:5433
mise run db-down            # stop the repo-local Postgres database
mise run dev                 # run API + client dev servers
mise run build               # build API + client
mise run check               # run API + client checks
mise //apps/api:build        # build apps/api/bin/api
docker build -f apps/api/Dockerfile -t gosvelteboilerplate-api .
mise //apps/api:dev          # run only the API
mise //apps/client:dev       # run only the frontend
mise //apps/client:build     # build only the frontend
```

If your local `mise` install does not pick up monorepo tasks automatically, enable experimental features in your shell before running commands:

```sh
export MISE_EXPERIMENTAL=1
```
