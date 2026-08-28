# Sonde — implementation plan

Self-hosted uptime monitoring for the Facile Suite. Uptime Kuma alternative, suite-native:
porte auth, Antenne eventing, muse UI, tronc chassis. This plan is a cold-start handoff —
a fresh reader needs no prior conversation.

## Goal

Ship Sonde v0.1.0: HTTP(S)+TLS / TCP / push-heartbeat monitors, incidents, uptime percentages,
public per-project status pages, Antenne alerting via enveloppe events, porte auth
(SSO + local fallback), RGPD-compliant retention — plus a companion CLI repo
(`FacileStudio/sonde-cli`) per the suite CLI standard.

## Why (evidence)

The suite has no monitoring story: Nook aggregates events, Journal logs, Antenne alerts, but
nothing watches whether the services themselves are up. Uptime Kuma is the incumbent
alternative and fails the suite's needs: single-user, Node-heavy, no event bus, and its
unpruned check history balloons Postgres (documented 5GB→2.7MB recoveries in the field).
Sonde reuses the suite's existing plumbing instead of rebuilding Kuma's notification zoo.

## Approach

Fork `Sonde` (already cloned at `~/Projects/Facile/Code/Sonde`, pre-rename),
run `scripts/rename.sh Sonde`, then build the Go-family standard app: tronc middleware +
error envelope, tronc/migrate goose migrations, porte (`oidc` + `local` kits on one session
manager), GORM + Postgres 16, SvelteKit 5 + muse client. The only genuinely new code is the
checker engine: a bounded-concurrency goroutine pool running hard-timeout-limited probes on
per-monitor tickers, writing raw checks that a daily pruner rolls up into hourly/daily
aggregates. Events go out through `antenne-client` (Go) wrapped in the `enveloppe` contract,
which this plan extends with a `monitor` object type and the `sonde` app name.

## Decisions already made (do not re-litigate)

| Topic | Decision |
|---|---|
| Auth | porte `oidc` + `local`; production deploys set `SSO_ONLY=true`, local passwords remain available for self-hosters without an IdP |
| Alerts | No notification channels in-app. Emit `monitor.down/up` events to Antenne via `antenne-client`; fan-out is Antenne's job |
| Declarative setup | Env vars for instance config (tronc standard). `GET /api/config/export` snapshot in v1. Seed-file import deferred (v1.5). No live YAML config — files describe the binary, database describes the world |
| Retention | Raw checks 30 days (env-tunable), hourly rollups 13 months, daily rollups indefinite. Prune job runs daily, documents itself in `docs/RETENTION.md`. Aggregates carry no personal data (storage-limitation clean) |
| Status pages | Public, unauthenticated, slug-based routes served by the same binary. Multiple pages, each selecting a set of monitors |
| ICMP ping | Deferred — raw sockets/CAP_NET_RAW pain in containers |
| TLS expiry | Recorded during HTTPS checks; incident/warning when < 21 days (configurable), matching common practice |

## Steps (ordered)

### Track A — scaffold & identity (blocks everything)

1. `~/Projects/Facile/Code/Sonde` — run `sh scripts/rename.sh Sonde` on the clean tree.
   Verifies module path becomes `github.com/FacileStudio/Sonde/apps/api`, dev DB name/port
   rewritten. [module-path]
2. Repo root — enable hooks: `git config core.hooksPath .githooks` (or `mise run hooks`),
   `mise trust && mise install && mise run bootstrap`. Exit criterion: `mise run check`
   passes untouched. [filet]
3. Create GitHub remotes `FacileStudio/Sonde` and `FacileStudio/sonde-cli`, push initial
   commit. Register both in `facile/internal/manifest/tools.yml` (sonde-cli row) once the CLI
   exists — not before. [distribute]

### Track B — domain core (parallel with D, blocks C/E/F)

4. `apps/api/migrations/00001_sonde.sql` — tables: `monitors` (id, owner_id, slug, name,
   type[http|tcp|push], target, interval_seconds, timeout_seconds, expected_status,
   expected_keyword, enabled, push_token unique, created_at, updated_at), `checks_raw`
   (monitor_id, ok, latency_ms, error, created_at), `checks_hourly` / `checks_daily`
   (monitor_id, bucket_start, total, failed, avg_latency_ms), `incidents` (monitor_id,
   opened_at, resolved_at, cause), `status_pages` (slug, title, created_at),
   `status_page_monitors` (page_id, monitor_id, position). [migrations]
5. `apps/api/internal/monitors/` — domain types, CRUD store over GORM, validation per type.
6. `apps/api/internal/checker/` — the engine. Bounded worker pool; one ticker per enabled
   monitor; `context.WithTimeout` around every dial/request (a hung probe must never leak);
   N=3 consecutive failures opens an incident, first success resolves; TLS cert expiry read
   during HTTP probes; writes `checks_raw`, updates rollups, emits state transitions on a
   channel consumed by the event publisher. Testable with `httptest` servers.
7. `apps/api/internal/checker/push.go` — `POST /api/push/{token}` route (unauthenticated by
   design — the token IS the credential): records a heartbeat; silence past
   `interval × grace` marks the monitor down.
8. `apps/api/internal/retention/` — daily job: prune `checks_raw` older than
   `SONDE_RAW_RETENTION_DAYS` (default 30), prune hourly rollups older than 13 months,
   aggregate into hourly/daily buckets. Writes nothing personal to aggregates.
9. `apps/api/internal/uptime/` — percentage queries off rollup tables (24h / 7d / 30d / 90d).

### Track C — API surface (needs B)

10. `apps/api/internal/httpapi/` — chi routes wired through tronc: monitors CRUD
    (auth-required, `X-Facile-CSRF` on cookie mutations), incidents list, uptime stats,
    `GET /api/config/export` (JSON snapshot of monitors + status pages, no secrets),
    public status-page routes `GET /status/{slug}` returning JSON for the client renderer.
    OpenAPI registry entries so `/docs` stays honest. [auth/porte]

### Track D — porte adoption (parallel with B)

11. `apps/api/cmd/api/main.go` + `apps/api/internal/database/` — replace boilerplate
    `authcrypto`/`authcontext` local-auth with porte: `porte/session` session manager,
    `porte/pg` stores, mount `porte/local` (register/login) AND `porte/oidc` (PKCE flow).
    Read `~/.mycelium/memory/bugs/porte-adoption-null-oidc-subject.md` before wiring — the
    traps are documented there even though greenfield means no legacy rows. Floor: current
    porte ≥ v0.2.10 semantics. [auth/porte]
12. `.env.example` + `docs/` — document `SSO_ONLY=true`, `OIDC_ISSUER`,
    `OIDC_CLIENT_SECRET`, `TRUST_EMAIL_WITHOUT_VERIFIED_CLAIM` unset. Login page renders the
    `?error=` redirect parameter outside any SSO_ONLY gate. [auth/porte]
    **Superseded on the provider**: this step assumed Authentik, which hardcoded
    `email_verified:false`. The provider is now Registre at `https://sso.facile.studio`,
    which reports the claim truthfully. Keep the flag unset anyway. See `docs/AUTH.md`.
13. Delete `apps/api/internal/authcrypto/`, `ticketcode/` if unused after the swap — deletion
    over addition.

### Track E — events (needs B; coordinate with enveloppe repo)

14. `enveloppe` repo (cross-repo PR) — add `sonde` to the app list, add `monitor` object type
    + payload shape (monitor_id, facile_id, slug, name, status, latency_ms, error), additive
    only (no envelope version bump — repo precedent: Mycelium addition kept `EventVersion`).
    Both TS and Go halves, identical JSON keys. [events]
15. `apps/api/internal/events/` — `antenne-client` (Go, pinned
    `github:FacileStudio/antenne-client` commit) emitting `FacileEvent` envelopes on incident
    open/resolve. Actor identity: the monitor's owner email (monitors always have an owner;
    solves the `actor_email` contract requirement machine-actors otherwise break). Disabled
    cleanly when `ANTENNE_INSTANCE` unset. [events] [distribute]

### Track F — client UI (needs C/D)

16. `apps/client/src/routes/` — login (porte `/auth/config`-aware), dashboard (monitor list
    with live status dots + uptime %), monitor create/edit forms per type, incident timeline,
    status-page builder (pick monitors, set slug/title), public `/status/[slug]` renderer.
    muse tokens throughout, Svelte 5 runes, mobile-first, fetch wrapper sends
    `X-Facile-CSRF` on writes. [muse] [auth/porte]

### Track G — sonde-cli (independent; starts after Track A)

17. New repo `FacileStudio/sonde-cli` — **Go** (suite's new CLI default; joins
    `mycelium`, `antenne-cli`), cobra or stdlib `flag` per existing Go CLIs' shape, per
    CLI-STANDARD §6: `sonde login` (device flow via porte), token in keyring/config per
    standard, commands: `monitors list/add/remove`, `push <token>` (the cron-friendly
    heartbeat — curl-free scripting), `status`, `incidents`. Releases via GoReleaser,
    registered in `tools.yml`, installable through `facile`.

## Files to Modify / New

- `scripts/rename.sh` output across repo — module path, DB identity (Track A)
- `apps/api/go.mod` — adds porte, antenne-client, enveloppe deps; removes dead boilerplate deps
- `apps/api/migrations/00001_sonde.sql` — new, all core tables
- `apps/api/internal/{monitors,checker,retention,uptime,httpapi,events}/` — new packages
- `apps/api/cmd/api/main.go` — porte + tronc + checker + pruner wiring
- `apps/client/src/routes/**` — dashboard, editors, status pages
- `docs/RETENTION.md` — documented periods + RGPD reasoning (storage limitation)
- `enveloppe/go/…`, `enveloppe/ts/…` — contract extension (separate repo, separate PR)
- `FacileStudio/sonde-cli/**` — new repo, Go

## Exit criteria

- `mise run check` passes; `filet` gate clean without threshold edits [filet]
- Checker integration test: httptest server flapped down/up opens then resolves an incident;
  a handler that hangs past timeout does not leak a goroutine (count stable across intervals)
- Push monitor: heartbeat keeps it green; silence past grace flips it red and emits an event
- Retention job: raw rows beyond 30 days gone, rollups intact, verified in test DB
- Cookie-authenticated mutation without `X-Facile-CSRF` returns 403 while reads succeed —
  tested over the browser transport, not just bearer [auth/porte]
- `SSO_ONLY=true`: password form hidden, OIDC callback lands authenticated;
  `SSO_ONLY=false`: both paths land on the same session
- Export endpoint returns import-shaped JSON with zero secrets in output
- Public status page reachable logged-out; private routes are not
- enveloppe PR merged; emitted JSON deserializes in both language implementations

## Risks / unknown unknowns

- **enveloppe is a contract repo** — adding `sonde` touches every consumer; version bump and
  both-language parity must land before Track E ships, else events are unparseable elsewhere.
- **actor_email for machine-originated events** — solved by attributing to monitor owner, but
  if ownership transfers, historical attribution shifts. Acceptable; noted.
- **Checker load** — interval floor (e.g. 20s minimum) and a global cap on concurrent probes
  prevent a user with 500 monitors from DDoSing the VPS hosting Sonde.
- ~~**Authentik registration**~~ **Registre registration** — Authentik is gone and
  `porte.facile.studio` 404s. Sonde registers as client_id `sonde` against Registre, which
  serves one issuer at the root: `https://sso.facile.studio`. Infra step, easy to forget.
- **Rollup correctness under clock skew / missed tickers** — aggregate from raw on read for
  the recent window if needed; don't let a gap look like 100% uptime.

## Skip (YAGNI)

- ICMP/ping monitors — container socket pain, TCP covers most
- Multi-region distributed probes — architecture nobody needs before single-node works
- Browser transaction monitoring — drags Chromium in, never
- In-app notification channels (Telegram/Discord/email…) — Antenne exists
- Live YAML/file config, last-write-wins sync — two-master drift machine; export + future
  idempotent seed covers declarative setup
- Maintenance windows, SLA reports, DNS checks — post-v1 backlog, not planned here
- Docker/K8s/host monitoring — different product (Vision territory)

---

*Checked against: migrations (tronc/migrate goose), auth/porte floor (CSRF, SSO_ONLY,
email_verified trap), muse, module-path, filet, events (enveloppe/actor_email), distribute
(github: pseudo-version pins), CLI-STANDARD (§6 credentials, tools.yml manifest).*
