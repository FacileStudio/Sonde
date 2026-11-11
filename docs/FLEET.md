# Fleet maintenance

Sonde keeps one picture of every repository: open vulnerabilities, version drift, CI state.
It stores what a scheduled runner reports and ranks it. It runs no scanner and touches no
repository.

```text
scheduled runner -> POST /api/fleet/reports -> fleet_* tables -> /api/fleet/actions, /drift, /graph, /conformance, /projects
```

## The runner

`apps/api/cmd/fleetscan` runs nightly from `.github/workflows/fleet-scan.yml`. It lists the
organisation's repositories, reads each default branch's tree, fetches the files douane needs
through the contents API, runs `douane sweep` over them, and delivers one gzip report. It never
clones and never runs a repository's code. Archived repositories are sent as not covered with
the reason `archived` and are never treated as vanished.

It fetches every lockfile douane recognises, including the ones it cannot parse (`yarn.lock`,
`pnpm-lock.yaml`, `requirements.txt`, `.csproj` and the rest of douane's list), so douane can
raise the gap instead of the repository reading as clean. It also fetches `package.json`
files, workflow files and Dockerfiles. Each repository is written under its numeric GitHub
id, so a repository named `.github` is not skipped as a hidden directory.

Measured on 2026-11-10 against FacileStudio: 88 repositories listed, 79 scanned, 9 with nothing
to scan, 75 seconds, 676 API calls, a 3 MB report. Two consecutive runs ingested with 0 new and
0 resolved findings.

Coverage decides whether Sonde may resolve a project's findings:

| Repository | `covered` | Counted in |
|---|---|---|
| No lockfile at all, or an empty repository | yes, with no findings | expected and scanned |
| Scanned with no blocking gap | yes | expected and scanned |
| douane `failed`, or a blocking gap (below) | no, `not_covered` names the gap kinds | expected |
| Tree or files could not be fetched, more than 80 manifests, or douane returned nothing | no | expected |
| Its data fails the server's `Validate` | no, `not_covered` is `invalid scan data` | expected |

A gap blocks coverage only when it can hide a finding. Resolving a project's findings after such
a scan would close flaws that were never checked again. Blocking gaps:

- an `unreadable` lockfile, one douane supports but could not read;
- an `unresolved` advisory, one whose OSV record could not be fetched;
- an `upstream` failure of OSV itself;
- an `unsupported` lockfile, but only when douane read no packages at all from the repository,
  so a yarn-only repository never passes as clean.

These gaps do not block:

- a `severity` or `range` gap, which only qualifies a finding douane did report;
- an `upstream` gap on the `kev` or `epss` feed, which only costs the ranking;
- an `unresolved` Dockerfile, whose Go module douane could not name while the module's own
  `go.mod` was read;
- an `unsupported` lockfile beside packages douane did read. It was never readable, so no finding
  from it exists to be wrongly resolved. For example, Opus's `pnpm-lock.yaml` sits beside its
  readable `go.mod`. The npm side of such a repository is not scanned at all.

A blocking gap at the sweep level comes from the shared OSV resolution and cannot be pinned on
one repository, so it withholds every repository that has packages.

- The report is checked with the server's own `Validate` before delivery. One invalid project
  is demoted to not covered rather than failing the whole night.
- Delivery is retried twice on a network error or a 5xx, never on a 4xx.
- The runner exits non-zero after delivering an incomplete run, so the workflow skips the
  heartbeat and two incomplete nights raise an incident.
- GitHub requests honour `Retry-After` and `X-RateLimit-Reset` (waiting at most 60 seconds) and
  fail rather than truncate a response over 64 MB.
- The log prints counts and exit codes only. Findings, douane's stderr and Sonde's error text
  never reach the workflow log, which is public.
- A pin's `resolved` value is the git ref after `#` in `package.json` (a `semver:` prefix is
  dropped) or the version in `go.mod`. It is compared literally with the newest tag, so a
  branch or a range always reads as drift. `replace` directives are ignored.
- Filet counts as wired when a workflow uses `FacileStudio/filet@…` or runs `filet check` or
  `filet test`.
- It also fetches `docker-compose.yml` (and `compose.yml`), every `.env.example`, `filet.yml`,
  `lefthook.yml` and `scripts/check.sh`. These are read for the graph and the checks below only:
  they are never written to douane's tree and do not count towards the 80-manifest limit.

## Edges

The runner derives each covered repository's dependencies from the files it fetched. An edge is
`{to, kind, auth, evidence}`; the server stores it under the reporting project as `from`.

| Kind | Read from | `to` | `evidence` |
|---|---|---|---|
| `lib` | every `github.com/FacileStudio/<x>` module in a `go.mod` and every `@facile/<x>` package in a `package.json`, tracked or not | the listed repository named `<x>`, ignoring case | the pin spec |
| `call` | a variable `<APP>_URL`, `_API_URL`, `_AUDIENCE`, `_API_TOKEN`, `_SERVICE_KEY`, `_INTEGRATION_KEY` or `_WEBHOOK_SECRET`, where `<APP>` is a listed repository's name upper-cased with dashes as underscores | that repository | the variable |
| `event` | `ANTENNE_URL` or `ANTENNE_INSTANCE` | `Antenne` | the variable |
| `identity` | `OIDC_ISSUER` | `Registre` | the variable |
| `logs` | `JOURNAL_URL` | `Journal` | the variable |

A call edge carries `auth`: `url` for `_URL` and `_API_URL`, `audience` for `_AUDIENCE`,
`static_key` for `_API_TOKEN`, `_SERVICE_KEY` and `_INTEGRATION_KEY`, and `webhook` for
`_WEBHOOK_SECRET`. The graph draws `static_key` and `webhook` calls in the warning tone.

Variables come from assignments in `.env.example`, commented examples included, and from the keys
and `${...}` interpolations of a compose file outside its comments. An edge to the repository
itself is dropped.

## Conformance checks

Each covered repository gets a list of `{id, ok, detail}`. A check that does not apply is left out,
never failed.

| Check | Passes when | Asked of |
|---|---|---|
| `gate.filet` | a workflow, `lefthook.yml` or `scripts/check.sh` uses `FacileStudio/filet@` or runs `filet check` or `filet test` | every repository |
| `gate.douane` | the same places use `FacileStudio/douane@` or run `douane scan` or `douane sweep` | every repository |
| `gate.tests` | the same places run `go test`, `bun test`, `cargo test` or `vitest` | every repository |
| `gate.race` | a `go test` line there carries `-race` | repositories with a `go.mod` |
| `hooks.lefthook` | `lefthook.yml` sits at the root | every repository |
| `license.present` | a `LICENSE`, `LICENCE` or `COPYING` file sits at the root | every repository |
| `migrations.sql` | a `migrations/*.sql` path exists | repositories with `apps/api/go.mod` |
| `docker.single` | exactly one Dockerfile exists, at the root | repositories with `apps/api` |
| `pins.current` | every tracked pin equals the newest tag in `latest` | repositories with a tracked pin whose newest tag is known |
| `fuzz.present` | a `testdata/fuzz/` or `fuzz/` directory is committed | repositories with a `go.mod` or `Cargo.toml` |

Paths under `node_modules/` and `vendor/` are ignored for every path check.

## The graph and conformance routes

- `GET /api/fleet/graph` returns `{nodes, edges}`. A node is
  `{id, name, tier, deployed_version, latest_tag, uptime_status, checks_ok, checks_total, open_act}`.
  `id` is 0 for a target edges name but the runner never listed. `latest_tag` is known for tracked
  libraries only. `uptime_status` is `up`, `down` or `unknown` from the newest check of the enabled
  monitors whose name equals the project's, ignoring case, and empty when no monitor has that name.
  `open_act` counts the `act` actions that touch the project.
- `GET /api/fleet/conformance` returns `{checks, projects}`: every check id seen, and per project
  its checks with `passed` and `total`, the smallest passing share first.

Both sit behind the same gate as every other read route. A covered project's edges and checks are
replaced wholesale on each delivery, inside the ingest transaction. A project that is not covered
keeps the rows of its last covering scan.

## Deploy order

The report stays at `schema_version` 1: `edges` and `conformance` are additive and omitted when
empty. The server decodes reports strictly (`DisallowUnknownFields`, through tronc's
`httpjson.DecodeGzipJSONLimit`), so a server that predates them refuses a report that carries
them with 400, and the runner does not retry a 4xx. Deploy the server before the workflow runs
the new runner. The workflow builds the runner from the default branch, so merge and deploy in
the same window, before the next night.

| Flag or variable | Default | What it does |
|---|---|---|
| `-org`, `FLEET_ORG` | `FacileStudio` | Organisation to scan |
| `-github-api`, `GITHUB_API_URL` | `https://api.github.com` | GitHub API base URL |
| `-douane`, `DOUANE_BIN` | `douane` | Path to the douane binary |
| `-workers` | `8` | Repositories fetched in parallel |
| `-out` | empty | Write the report as JSON to this file instead of delivering it |
| `GITHUB_TOKEN` | required | App installation token |
| `SONDE_URL`, `SONDE_FLEET_TOKEN` | required unless `-out` | Where to deliver, and the bearer |

A dry run:

```sh
cd apps/api && go build -o /tmp/fleetscan ./cmd/fleetscan
GITHUB_TOKEN=$(gh auth token) DOUANE_BIN=$(which douane) /tmp/fleetscan -out /tmp/report.json
```

## Setting it up

1. Register a GitHub App on FacileStudio. Repository permissions: Contents read, Actions read,
   Metadata read. No webhook, no user authorization, no device flow. Install it on all
   repositories, so a new repository cannot escape the scan.
2. On the Sonde deployment, set `SONDE_FLEET_TOKEN` (32 or more random characters) and
   `SONDE_FLEET_ADMINS`. Use the `dokploy-env-safe` flow: a partial env update drops keys.
3. In Sonde, create a push monitor named `fleet-scan` with an interval of 13 hours (46800
   seconds). Its push token is the heartbeat. Sonde calls a push monitor down after twice its
   interval without a heartbeat, so 13 hours raises an incident after 26 hours of silence. The
   interval cannot exceed 24 hours, and 24 would wait 48 hours, letting a missed night pass
   unnoticed.
4. In the Sonde repository settings, add the secrets `FLEET_APP_CLIENT_ID` (the App's Client
   ID, shown on its settings page), `FLEET_APP_PRIVATE_KEY`, `SONDE_FLEET_TOKEN` and
   `FLEET_HEARTBEAT_TOKEN`, and the variables `SONDE_URL` (`https://sonde.facile.studio`) and
   `FLEET_SCAN_ENABLED` (`true`). Keep a copy of the App key in Casier.
5. For the Rescan button on the Maintenance page, create a fine-grained personal access token
   (https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens#creating-a-fine-grained-personal-access-token)
   with resource owner FacileStudio, repository access "Only select repositories: Sonde", and
   one repository permission: Actions, read and write. Set it as `SONDE_FLEET_DISPATCH_TOKEN` on
   the Sonde deployment with the `dokploy-env-safe` flow, and keep a copy in Casier. The button
   calls `POST /api/fleet/scans`, which queues the workflow through GitHub's dispatch API
   (https://docs.github.com/en/rest/actions/workflows#create-a-workflow-dispatch-event), at most
   once every five minutes. Without the token the route answers 412 and the page shows why.
   The queued run still needs step 4: without the App it completes as skipped.
6. Run the workflow once by hand. From then on, a night that fails or delivers an incomplete run
   sends no heartbeat, and 26 hours without one opens a Sonde incident on `fleet-scan`.

The workflow pins douane by version and by the sha256 of its Linux archive, and every action
by commit. To move douane, change both `DOUANE_VERSION` and `DOUANE_SHA256`, taking the hash
from the release you reviewed.

## When fleet-scan opens an incident

The incident means no heartbeat arrived for 26 hours: the workflow did not run, failed, or
delivered an incomplete run.

1. Check whether the workflow is still enabled: `gh workflow view fleet-scan.yml -R
   FacileStudio/Sonde`. GitHub disables scheduled workflows in a public repository after 60 days
   without repository activity
   (https://docs.github.com/en/actions/managing-workflow-runs/disabling-and-enabling-a-workflow).
2. Re-enable it if needed: `gh workflow enable fleet-scan.yml -R FacileStudio/Sonde`.
3. Run it now: `gh workflow run fleet-scan.yml -R FacileStudio/Sonde`, then read the run's exit
   status with `gh run list --workflow fleet-scan.yml -R FacileStudio/Sonde`. A run that delivered
   but was incomplete exits non-zero; the Maintenance page shows which repositories were not
   covered and why.

## Archived repositories

The runner reports an archived repository as not covered with reason `archived`. Ingest moves
such a project to the `archive` tier, so its frozen findings rank as track and leave the
inbox and the drift view. Unarchiving it does not restore the old tier: set it again on the
Projects tab.

## Who can see it

Fleet data is a ranked list of weaknesses, so access is narrower than the rest of Sonde. A
read route needs both `is_admin` and an email in `SONDE_FLEET_ADMINS`. The first account to
sign in becomes admin, and erasing the last admin promotes the lowest-numbered user, who
under public sign-up can be a stranger. An empty `SONDE_FLEET_ADMINS` closes every read route.

The report route is outside the session gate. Its caller is a job holding `SONDE_FLEET_TOKEN`.
Empty turns ingest off.

## The report

`POST /api/fleet/reports`, gzip-compressed JSON, `schema_version` 1. The types are in
`apps/api/internal/fleet/types.go`; the runner builds the same ones.

- **`covered`** says whether the runner scanned the project. A project that is not covered
  keeps its findings. Absence from a scan that did not happen is not evidence a flaw is gone.
- **`repos_expected` and `repos_scanned`** make a short run visible. The run is stored as
  incomplete.
- **Finding keys** are derived by the server from the smallest name in the alias set, so a
  flaw that flips between its CVE and its GHSA name between nights is neither resolved nor new.
  A finding also keeps its key when a new alias appears, for example a CVE assigned after its
  GHSA.
- **A complete run** (`repos_scanned == repos_expected > 0`) also resolves the findings of
  projects it no longer lists, and marks them not covered. A complete run that lists far fewer
  repositories than are currently covered (below 90%, or more than 5 fewer) is stored as
  incomplete with `coverage_dropped` and resolves nothing, and the server logs a warning, so a
  narrowed App installation cannot empty the inbox. When repositories really were deleted or
  transferred, run the workflow once by hand with `accept_shrink` checked (`fleetscan
  -accept-shrink`, or `FLEET_ACCEPT_SHRINK=true`).
- **Deliveries are serialised.** A report that is not newer than the last stored one is refused
  with 409, so a retried delivery is safe to repeat.
- **Logs.** The server logs every delivery at info (run id and counts), refused deliveries at
  warn with their code, internal failures at error with their cause, and failed token attempts
  at warn with the remote address, never the token.
- **Rate limit.** Accepted deliveries are limited to a burst of ten, refilled at one a minute.
  The limit sits after the token check, so a stranger cannot spend the runner's budget.

## Ranking

Findings group by `ecosystem|package|fixed_in`. Each group gets one level:

1. Only archived or experimental projects: `track`, whatever the score.
2. Known exploited, or EPSS of 0.1 or more: `act`.
3. Critical in a deployed project: `act`. High in a deployed project: `schedule`.
4. Critical elsewhere: `schedule`.
5. Everything else: `track`.

A new project is `unclassified`, so it cannot reach `act` on severity alone. Set tiers with
`PATCH /api/fleet/projects/{id}` before trusting the order.

## Holds

A hold keeps a package on a version on purpose. It needs a reason and an end date within a
year. When it lapses the project reappears as behind.

## Retention

Resolved findings and runs are kept for 90 days.
