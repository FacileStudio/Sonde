# Fleet maintenance

Sonde keeps one picture of every repository: open vulnerabilities, version drift, CI state.
It stores what a scheduled runner reports and ranks it. It runs no scanner and touches no
repository.

```text
scheduled runner -> POST /api/fleet/reports -> fleet_* tables -> /api/fleet/actions, /drift, /projects
```

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
