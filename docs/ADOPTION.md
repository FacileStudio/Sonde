# Adopting the shared suite pieces

The recipe a Go and SvelteKit suite app follows to move onto the libraries released on
2026-11-10. Sonde did it first, in v0.5.0. Each step is one commit, and the gate passes at each
commit. Every trap below is one Sonde hit.

| Step | Moves to | Replaces |
|---|---|---|
| 1 | tronc v0.21.0, porte v0.17.0, muse v0.17.1 | older pins |
| 2 | `porte/kit` | `modules/auth` glue, `internal/authcontext`, `internal/middleware/auth.go` |
| 3 | `antenne-client/go` v1.0.0 | the hand-written `internal/events` wrapper and `enveloppe/go` |
| 4 | `@facile/client` v0.1.0 | the fetch kernel, 401 redirect and theme store in `apps/client/src/lib` |
| 5 | douane 0.7.0, filet `preset: facile` | older gate pins |
| 6 | `lefthook.yml` | `.githooks/` or no tracked hook |
| 7 | AGPL-3.0-only | no licence |

## Before you start

1. Work in your own worktree, never in `~/Code/Facile/<app>`:
   `git -C ~/Code/Facile/<app> fetch origin && git -C ~/Code/Facile/<app> worktree add -b conv/adopt ~/.cache/facile-conv/<app>-adopt origin/<default>`.
2. Start your own Postgres:
   `docker run -d --name conv-<app> -e POSTGRES_PASSWORD=conv -p 127.0.0.1:<port>:5432 postgres:16`,
   then `export TEST_DATABASE_URL=postgres://postgres:conv@127.0.0.1:<port>/postgres?sslmode=disable`.
3. Run the gate on the untouched branch and keep the output: `sh scripts/check.sh` and
   `filet check > baseline-filet.txt`. Later you diff filet findings against this file.
4. Capture production before you push anything:
   `curl -s https://<app>.facile.studio/api/auth/config`, the `Location` of
   `GET /api/auth/oidc` (strip `state`, `nonce` and `code_challenge`), and the 401 body of one
   authenticated route.

**Trap: run one test process per database.** `tronc/testdb` truncates by prefix. Two `go test`
runs on the same `TEST_DATABASE_URL` at once, for example the gate in the background and a
targeted `-run` in the foreground, break each other's fixtures with foreign-key errors that
look like real failures. Run the gate alone before each commit.

## Step 1: bump the libraries

```sh
cd apps/api
GOFLAGS=-mod=mod GOPROXY=direct go get github.com/FacileStudio/tronc@v0.21.0 github.com/FacileStudio/porte@v0.17.0
go mod tidy
cd ../client
# set "@facile/muse": "github:FacileStudio/muse#v0.17.1" in package.json
bun install
```

Read each CHANGELOG between your pin and the new tag first
(`git show <tag>:CHANGELOG.md`). From tronc v0.17.1 and porte v0.13.0 every entry is additive.
Bump nested modules (`porte/directory`, `tronc/migrate`, `tronc/testdb`) only when the build
asks for it; Sonde needed none. Update the version numbers your `CLAUDE.md`, `docs/AUTH.md` and
`docs/architecture.md` quote.

`bun.lock` records muse by the annotated tag object's hash (`4c73965` for v0.17.1), not the
commit's (`4ab597b`). That is expected. The Docker client stage runs
`bun install --frozen-lockfile`, so commit the lockfile with `package.json`.

## Step 2: porte/kit

Read `docs/kit.md` in porte at v0.17.0. The Sonde commit (`refactor(api): replace the auth glue
with porte/kit`) is the worked example; diff it with `git show` and copy its shape.

1. Delete `internal/authcontext/`, `internal/middleware/auth.go`, `modules/auth/userstore.go`
   and the erase code. Keep the `directory_state` store: move it to
   `modules/auth/deletion_state.go` unchanged.
2. Add `modules/auth/store.go`: one constructor returning `&kit.Store{DB: sqlDB, Columns:
   kit.Columns{...}, EraseData: ...}`. `docs/kit.md` has the `Columns` per app. A GORM app
   passes `gormDB.DB()`.
3. Put every app-owned delete in `EraseData`. The kit deletes the `users` row after it, in the
   same transaction. Sonde also hands the admin role to the oldest remaining account there:

   ```go
   UPDATE users SET is_admin = true
    WHERE id = (SELECT min(id) FROM users WHERE id <> $1)
      AND NOT EXISTS (SELECT 1 FROM users WHERE is_admin AND id <> $1)
   ```

   The `id <> $1` matters: the row being erased still exists when `EraseData` runs.
4. Replace `authcontext.FromContext` with `kit.FromContext` and the middleware with
   `kit.RequireAuth(sessions, users)`. Pass the same store as `oidc.Deps.Users`, as
   `directory.Deletions.OnPersonDeleted` (`users.EraseAccount`) and for the picture backfill
   (`users.BackfillPictures`).
5. Mount with `kit.Mount(api, kit.Config{Sessions: sessions, OIDC: oidcKit})` inside
   `router.Route("/api", ...)`.

The kit keeps the rules Sonde already had: the first account is admin when `Columns.Admin` is
set, the email is rewritten only when the provider marks it verified, the 401 message for a
session whose user is gone is `invalid auth token`. Check your app's own rules against
`docs/kit.md` "Rules the store applies" before you delete them. An app that rewrote the email
on every sign-in changes behaviour here, by owner decision.

**Prove the routes did not move.** `apiref_test.go` only proves every route is documented. To
prove the route list and the unauthenticated status codes are identical, drop this test into
both the old tree (a detached worktree at the pre-kit commit) and the new one, run it, diff the
two files, and delete it:

```go
func TestZZRouteWalk(t *testing.T) {
	configs := map[string]func(*env.Config){
		"bare": nil,
		"oidc": configureIssuer,
		"oidc+cli": func(c *env.Config) { configureIssuer(c); c.Porte.CLIAudience = "facile-cli" },
	}
	var lines []string
	for name, configure := range configs {
		router, _ := liveRouter(t, configure)
		_ = chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
			path := route
			for strings.Contains(path, "{") {
				i, j := strings.Index(path, "{"), strings.Index(path, "}")
				path = path[:i] + "1" + path[j+1:]
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(method, strings.Replace(path, "/*", "/x", 1), nil))
			lines = append(lines, fmt.Sprintf("%s %s %s -> %d", name, method, route, rec.Code))
			return nil
		})
	}
	sort.Strings(lines)
	os.WriteFile(os.Getenv("ROUTEWALK_OUT"), []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}
```

`liveRouter` and `configureIssuer` are Sonde's test helpers (`auth_test.go`): a router over a
test database, and a fake discovery document so `oidc.New` boots offline. Copy them if your app
has none. Sonde's walk gave 167 identical lines across the three configurations.

**Trap: the proof commit carries a `replace`.** Sonde's proof branch pointed porte at a local
worktree. If you cherry-pick from any proof branch, delete the `replace` line and run
`go mod tidy`; `go.mod` must require the tag.

## Step 3: antenne-client v1

```sh
cd apps/api
GOFLAGS=-mod=mod GOPROXY=direct go get github.com/FacileStudio/antenne-client/go@v1.0.0
```

1. **Configuration.** Call `antenne.ConfigFromEnv("<app>")`. It reads `ANTENNE_URL`, or the
   older `ANTENNE_INSTANCE` (URL wins), plus `ANTENNE_APP`, `ANTENNE_SECRET` and
   `ANTENNE_INSTANCE_ID`. Treat `antenne.ErrNotConfigured` as "publishing off" and return nil;
   any other error fails boot, which is what a URL without a secret deserves. Set
   `config.Events.Emit` yourself: `ConfigFromEnv` leaves it empty.
2. **Migration.** Copy `outbox.Migration` verbatim to the next number, for example
   `migrations/00008_antenne_outbox.sql`
   (`cp $(go env GOMODCACHE)/github.com/\!facile\!studio/antenne-client/go@v1.0.0/outbox/migration.sql ...`).
   Add a test that the file equals `outbox.Migration`, so the copy cannot drift. No GORM struct:
   only the library touches the table, through `database/sql`.
3. **Enqueue in the state change's transaction.** Find the write that produces the event (in
   Sonde, opening and resolving an incident) and call `outbox.Enqueue(ctx, tx, channel, event,
   key)` inside it. A GORM app can run that write through `sqlDB.BeginTx` with plain SQL rather
   than digging a `*sql.Tx` out of a GORM transaction. Treat `outbox.ErrDuplicate` as success.
4. **Drain.** `outbox.NewDrainer(sqlDB, emit, ...)` with `emit` wrapping `client.EmitNow`, never
   `Emit`, and `go drainer.Run(ctx)`.
5. **Envelope and channels.** Replace `github.com/FacileStudio/enveloppe/go` with
   `github.com/FacileStudio/antenne-client/go/envelope`; field names and JSON are identical. Take
   channel names from `catalog` (`catalog.MonitorDown`, ...). Grep the whole module for
   `enveloppe`: Sonde's webhooks dispatcher imported it too.
6. **Compose and docs.** Add `ANTENNE_URL: ${ANTENNE_URL:-}` beside `ANTENNE_INSTANCE` in
   `docker-compose.yml`, then `docs/configuration.md`, `.env.example` and the env table in
   `CLAUDE.md`. A variable compose does not forward never reaches the container.

Traps:

- **Variables read inside a library are invisible to a source scan.** Sonde's
  `internal/env/compose_test.go` fails when the server reads a variable compose does not
  forward, by parsing `os.Getenv` calls. `ConfigFromEnv` reads them in the dependency, so the
  test stopped seeing them. List them explicitly in the test (`libraryVariables`).
- **The client still gives up.** antenne-client v1 stops reconnecting after 20 attempts (about
  7.5 minutes) and does not retry the first dial. Keep `antenne.WithMaxReconnect(math.MaxInt32)`
  and your own dial loop with backoff, or a suite redeploy ends publishing until a restart.
- **The drainer reports `ErrNotConnected` on every pass** while the bus is down. Filter it with
  `errors.Is(err, antenne.ErrNotConnected)` in `outbox.WithOnError`, or the logs fill up. The
  default backoff caps at one minute, which is also how long a queued alert can wait after the
  bus returns; Sonde uses `outbox.WithMaxBackoff(5 * time.Second)`.
- **Return a nil interface, not a nil pointer.** "Publishing off" must reach the write path as a
  nil interface. Sonde's `(*Publisher).Queue()` returns `nil` on a nil publisher, so an
  unconfigured instance queues nothing into a table nobody drains.
- **A failed query aborts the Postgres transaction.** Anything you look up while building the
  event inside the transaction (Sonde reads the owner's email) must succeed or fail the whole
  write. Return the error; do not swallow it and continue on an aborted transaction.
- **A "dropped" counter means nothing with an outbox.** Sonde replaced `dropped` with `pending`
  (unsent rows) in its settings response and page.

Prove it with tests: an incident queues its events in the same transaction; an enqueue that
fails rolls the incident back; no configuration queues nothing; the drainer sends in id order
and a refused emit leaves the row queued. Sonde's are in `apps/api/outbox_test.go` and
`apps/api/internal/events/publisher_test.go`.

## Step 4: @facile/client

```sh
cd apps/client
# add "@facile/client": "github:FacileStudio/facile-client#v0.1.0" to dependencies
bun install
```

In `src/lib/backend.ts`, delete the kernel (`apiFetch`, the 401 handler, `toError`) and write:

```ts
import { createApi, redirectToLogin } from '@facile/client';

const apiFetch = createApi({ base: backendBaseUrl, onUnauthorized: redirectToLogin });
```

Keep the `backend` object and every method as it is. A third argument that meant "do not
redirect on 401" becomes `{ quiet: true }`. `ApiError` carries `status`, so call sites reading
`(err as { status?: number }).status` keep working.

Replace `theme.svelte.ts` with:

```ts
import { createTheme } from '@facile/client/theme';

export type { Theme as ThemeMode } from '@facile/client/theme';

export const theme = createTheme('<app>-theme');
```

Then `restoreTheme()` becomes `theme.restore()` and `setTheme(x)` becomes `theme.set(x)`.

Traps:

- **Pass your existing `localStorage` key** to `createTheme`, or every user's stored choice resets.
- A public route called through `apiFetch` should pass `{ quiet: true }`. Its error message is now
  the server's envelope message, not your old hard-coded fallback.
- `bun run check`, `bun run build` and the message check must pass. There is no client test
  runner; open the built SPA from the Docker image (below) to see it load.

## Step 5: gate pins

- **douane.** Wherever it is pinned (Sonde: `.github/workflows/fleet-scan.yml`), change the
  version and the archive's sha256 together. Take the hash from the release's
  `checksums.txt` and check it against the downloaded archive:
  `gh release download v0.7.0 -R FacileStudio/douane -p 'douane_0.7.0_linux_amd64.tar.gz' -p checksums.txt && sha256sum douane_0.7.0_linux_amd64.tar.gz`.
  For linux_amd64 it is `718745c1440905ff9c87c1d5eef866d3994beaec39e92f3c033835016f4d7dbb`.
  0.7.0 adds `permitted` to its JSON and can exit 3 on an invalid `douane.yml` in a sweep;
  check that whatever parses its output accepts both.
- **filet.** Check `git ls-remote --tags https://github.com/FacileStudio/filet v0.19.0`. When it
  exists: reduce `filet.yml` to `preset: facile` plus the exceptions the app really needs, run
  the v0.19.0 binary before and after, prove the finding lists are identical, and pin the
  workflow to v0.19.0. Never loosen `filet.yml` to pass. When Sonde adopted, v0.19.0 was not
  published, so Sonde's filet is unchanged; its CI still uses the filet action at v0.3.0.

## Step 6: lefthook.yml

Copy the shape Casier, Journal and Sablier share:

```yaml
remotes:
  - git_url: https://github.com/FacileStudio/hooks
    ref: v1
    refetch_frequency: 24h
    configs:
      - common.yml

pre-push:
  jobs:
    - name: check
      run: sh scripts/check.sh
```

Pin `lefthook = "2.1.12"` in `mise.toml` `[tools]` and run `mise lock lefthook` (only when
`lockfile = true`). Point the `hooks` task at `lefthook install --reset-hooks-path`, so a clone
still configured with `core.hooksPath .githooks` moves over, and delete `.githooks/`. Check the
file with `lefthook validate`.

**Trap: do not run `lefthook install` in a worktree.** It writes into the shared `.git/hooks`
of the owner's checkout.

## Step 7: licence

Follow "Licence rollout" in the suite convergence plan
(`docs/plans/suite-convergence.md` in the coordinator's checkout): the authorship gate first,
then the canonical AGPL text verbatim to `LICENSE`
(`curl -fsSL https://www.gnu.org/licenses/agpl-3.0.txt`, sha256
`0d96a4ff68ad6d4b6f1f30f713b18d5184912ba8dd389f86aa7710db079abcb0`),
`"license": "AGPL-3.0-only"` in every named `package.json`, a README "License" section with the
commercial-licence sentence, a CHANGELOG line, and the bounded dependency check. One commit:
`chore(license): add the AGPL-3.0-only licence`. Skip the licence and report it when the
authorship gate finds a human author other than the owner.

## Before pushing

1. `docker build -t <app>-adopt .` from the repo root.
2. Run it with OIDC unset: `docker run -d --network host -e PORT=<p> -e APP_ENV=development -e
   DATABASE_URL=<scratch db> <app>-adopt`. Check `/api/health` 200, `/docs` 200, `/` 200 HTML,
   and an authenticated route answering 401 JSON.
3. Run the new migration Down and Up by hand on a scratch database. The app binary only knows
   `migrate up|status|version|baseline`, so use goose at the version `go.mod` pins:
   `GOOSE_DRIVER=postgres GOOSE_DBSTRING=<scratch db> GOOSE_MIGRATION_DIR=migrations go run github.com/pressly/goose/v3/cmd/goose@v3.27.3 down`, then `up`.
4. Run the Go tests from a fresh clone of your branch with `-count=1`, so a fixture that
   `.gitignore` hides fails here and not in CI.

## Release and verify

Follow the app's release convention (CHANGELOG, `VERSION`, `chore: release vX.Y.Z`), push by
fast-forward, tag, push the tag, and watch CI. Then wait for production to serve the version
(`/version`), run `scripts/verify-deploy.sh`, and compare `/api/auth/config`, the OIDC redirect
and a 401 with what you captured before the push. If health, the login redirect or sign-in
fails because of your change, push revert commits at once. Never force-push.

Back to the [documentation index](README.md).
