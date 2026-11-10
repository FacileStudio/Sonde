# Fleet maintenance

Sonde keeps one picture of every repository: open vulnerabilities, version drift, CI state.
It stores what a scheduled runner reports and ranks it. It runs no scanner and touches no
repository.

```text
scheduled runner -> POST /api/fleet/reports -> fleet_* tables -> /api/fleet/actions, /drift, /projects
```

## The runner

`apps/api/cmd/fleetscan` runs nightly from `.github/workflows/fleet-scan.yml`. It lists the
organisation's repositories, reads each default branch's tree, fetches the lockfiles,
`package.json` files and workflow files through the contents API, runs `douane sweep` over
them, and delivers one gzip report. It never clones and never runs a repository's code.

Measured on 2026-11-10 against FacileStudio: 88 repositories listed, 79 scanned, 9 with nothing
to scan, 75 seconds, 676 API calls, a 3 MB report. Two consecutive runs ingested with 0 new and
0 resolved findings.

- A repository with no lockfile, or an empty one, has nothing to scan. It is reported, not
  expected.
- A repository whose tree or files could not be fetched, that has more than 80 manifests, or
  that douane returned nothing for, is reported as not covered and counts against the run.
- The log prints counts only. Findings never reach the workflow log.
- `-out report.json` writes the report instead of delivering it, for a dry run:

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
3. In Sonde, create a push monitor named `fleet-scan` with an interval of about 26 hours. Its
   push token is the heartbeat.
4. In the Sonde repository settings, add the secrets `FLEET_APP_ID`, `FLEET_APP_PRIVATE_KEY`,
   `SONDE_FLEET_TOKEN` and `FLEET_HEARTBEAT_TOKEN`, and the variables `SONDE_URL`
   (`https://sonde.facile.studio`) and `FLEET_SCAN_ENABLED` (`true`). Keep a copy of the App key
   in Casier.
5. Run the workflow once by hand. A missed night then opens a Sonde incident on `fleet-scan`.

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
