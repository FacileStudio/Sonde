# Client

SvelteKit frontend for the FDT monorepo.

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

The repo manages Node.js with `mise`, and the client dependencies are installed from `package-lock.json`.
