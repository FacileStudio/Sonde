# Client

SvelteKit frontend for the GoSvelteBoilerplate monorepo.

## Quick Start

From the repo root:

```sh
mise install
mise //apps/client:dev
```

If you are already in `apps/client`, you can use the local task form:

```sh
mise :dev
```

## Common Commands

```sh
mise //apps/client:install  # install dependencies
mise //apps/client:dev      # start the dev server
mise //apps/client:check    # run svelte-check
mise //apps/client:build    # create a production build
mise //apps/client:preview  # preview the production build
```

The repo manages `bun` with `mise`, and the client dependencies are installed from `bun.lock`
with `--frozen-lockfile`. There is no `package-lock.json` and no `npm` step.

## How this client is served

`adapter-static` builds a pure SPA into `build/`, which the Go API copies into its container
image and serves from `CLIENT_DIR`. Consequences:

- `src/routes/+layout.ts` sets `prerender = false` and `ssr = false`. There is no Node server
  at runtime, so nothing renders server-side.
- API calls are **relative to `/api`** (`src/lib/backend.ts`). Never an absolute host: the
  same build has to work in every environment, and in production the API is same-origin.
- In development, Vite proxies `/api` to `http://127.0.0.1:4000` (`vite.config.ts`), so the
  local request path matches production.
