# Retention

Sonde keeps raw probe results for 30 days, hourly rollups for 13 months, and daily rollups
forever. A background job in `apps/api/internal/retention/` builds the rollups and deletes
what is past its window, once every 24 hours.

Uptime Kuma is the cautionary tale this design exists to avoid: it stores every heartbeat
forever, so a busy instance grows an unbounded `heartbeat` table that nobody prunes until the
disk fills. Sonde aggregates on a schedule and deletes the detail behind it.

## The three tiers

| Table | What one row is | Kept for | Tuned by |
|---|---|---|---|
| `checks_raw` | One probe result | 30 days | `SONDE_RAW_RETENTION_DAYS` |
| `checks_hourly` | One monitor, one hour | 13 months | Not tunable, compile-time constant |
| `checks_daily` | One monitor, one day | Forever, never pruned | Not tunable |

The schema is in `apps/api/migrations/00001_sonde.sql`.

### What a raw check stores

`checks_raw` has `id`, `monitor_id`, `ok`, `latency_ms`, `error`, `tls_expires_at` and
`created_at`. The `error` column holds the probe's own failure text, so it can quote the
monitored host, its URL or its IP address. That is the row that carries detail, and that is
the row with the short window.

`checker.GormRecorder.RecordCheck` writes it, one row per probe, at each monitor's
`interval_seconds` cadence.

### What a rollup stores

`checks_hourly` and `checks_daily` have exactly five columns: `monitor_id`, `bucket_start`,
`total`, `failed` and `avg_latency_ms`. Four of the five are integers and the fifth is a
timestamp. No error text, no target, no URL, no address, no user.

## How a bucket gets filled

`checks_hourly` is written twice, on purpose.

The recorder increments it in the same transaction as the raw insert, so the dashboard shows
the current hour without scanning `checks_raw`. That counter is an approximation: the average
latency is folded in incrementally and a crash between probes can leave it short.

The retention job then rebuilds every hourly bucket before the current hour straight from
`checks_raw`, with `ON CONFLICT … DO UPDATE SET total = EXCLUDED.total`. The rebuild
overwrites rather than adds, so the daily run corrects whatever the incremental counter got
wrong, for as long as the raw rows behind it still exist. Once a bucket falls out of the raw
window it stops appearing in the rebuild query and is left alone.

`checks_daily` is built the same way, from `checks_hourly` buckets before today, weighting
each hour's average latency by its `total`.

## When the job runs

`main.go` starts it as `go pruner.Loop(backgroundCtx)`. `Loop` runs immediately at boot, then
every 24 hours from a `time.Ticker`. There is no cron entry and no separate container: the
API process owns the schedule.

Each run does three things in this order, and stops at the first error:

1. `rollupHours` rebuilds hourly buckets from `checks_raw`.
2. `rollupDays` rebuilds daily buckets from `checks_hourly`.
3. `prune` deletes raw rows past `SONDE_RAW_RETENTION_DAYS` and hourly rows past 13 months.

Aggregating before deleting is the whole point of the ordering. A run that pruned first would
throw away the detail it was about to summarize.

The ticker restarts with the process, so an API that redeploys more than once a day runs the
job on each boot. Every run logs `rolled up hourly checks`, `rolled up daily checks`,
`pruned raw checks` and `pruned hourly checks` with a `rows` count, so a run that did nothing
is visible in Journal.

## Tuning the raw window

`SONDE_RAW_RETENTION_DAYS` is the only retention knob. `internal/env/env.go` reads it and
falls back to `retention.DefaultRawRetentionDays`, which is 30.

The fallback is silent. Unlike `PORT` or `LOG_LEVEL`, a value that is not an integer or is
below 1 does not stop the process, it just becomes 30. Check the value you set, because
`SONDE_RAW_RETENTION_DAYS=thirty` boots happily on the default.

Lowering it shrinks the table and shortens how far back an operator can read individual
failures. It does not affect uptime percentages: `internal/uptime` answers the 24h, 7d, 30d
and 90d windows from `checks_hourly`, never from `checks_raw`. A 7-day raw window still gives
a correct 90-day uptime figure.

The 13-month hourly window is `hourlyRetentionMonths` in `retention.go`. Thirteen months
covers a year-over-year comparison plus one month of slack, and it keeps every query window
the API exposes answerable from a single table.

## Why this is storage-limitation clean

RGPD Article 5(1)(e) says personal data must be kept in an identifiable form no longer than
the purpose needs. Sonde's answer has two halves.

**The rollups hold no personal data.** A `checks_hourly` or `checks_daily` row is a monitor
id, a timestamp and three counters. Nothing in it identifies a person, and nothing in it can
be joined back to one except through `monitors.owner_id`, which is the account that created
the monitor and is retained for as long as the monitor is. Deleting a monitor cascades:
`checks_raw`, `checks_hourly`, `checks_daily` and `incidents` all declare
`REFERENCES monitors (id) ON DELETE CASCADE`, so removing a monitor removes its whole history
in one statement. Keeping counters indefinitely is therefore not indefinite retention of
personal data, it is retention of a count.

**The detail expires.** Everything that could carry more than a count lives in `checks_raw`,
whose `error` column is free text produced by whatever the probe hit. That table is the one
on a fixed, short, operator-tunable clock.

Sonde also monitors services, not people. A monitor's `target` is a URL or a host and port,
and the only account data in the database is the `users` row that owns it.

## The gap worth knowing about

**The retention job does not touch `incidents`.** An incident row carries `monitor_id`,
`opened_at`, `resolved_at` and a free-text `cause` written from the same probe output as
`checks_raw.error`. It survives long past the 30-day raw window and is only ever removed when
its monitor is deleted.

That is a small table, one row per downtime episode rather than one per probe, so it is not a
growth problem. It is a scope problem: the shortest-lived copy of the probe's error text has
a 30-day clock and the longest-lived copy has none. Give `incidents` a window before Sonde
monitors anything whose failure messages could quote a person.

## Fleet tables

The same daily job prunes the fleet tables after 90 days: resolved rows in `fleet_findings`
(by `resolved_at`) and rows in `fleet_runs`. An open finding is current state and is never
pruned, however old. `fleet_pins` and `fleet_latest` hold only the latest report, so they do
not grow.

Back to the [documentation index](README.md).
