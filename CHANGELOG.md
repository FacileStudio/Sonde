# Changelog

## [0.3.1] - 2026-11-10

### Security

- `PATCH /api/users/me` no longer changes the email. The fleet allow-list could otherwise be met by a promoted account setting its email to a freed allow-listed address. The email comes only from Registre, refreshed on sign-in when the address is verified.

### Fixed

- The fleet runner no longer reports a repository as covered when douane could not read it, so Sonde never resolves findings that were not checked again. Only a gap that can hide a finding withholds coverage: an unreadable lockfile, a failed advisory fetch, an OSV failure, or an unsupported lockfile in a repository douane read nothing else from.
- A finding keeps its key when a new alias appears, such as a CVE assigned after its GHSA, and blank aliases are ignored.
- A complete run resolves the findings of repositories it no longer lists.
- Fleet deliveries are serialised, and a report not newer than the last stored one is refused with 409.
- Fleet reports with NUL bytes are refused, and an empty run is never marked complete.
- Fleet pruning runs even when checks pruning fails.
- The maintenance page saves one role change at a time and re-ranks after it. It translates severities, CI states and scan states, shows hold reasons as text, and lets every repository behind "+N more" be reached.

### Changed

- The nightly workflow uses douane 0.6.2, verified against a pinned sha256. Its actions are pinned to commits, and the App token is read-only.
- The runner reads douane's full lockfile list, validates each project before delivering, and retries delivery on network errors and 5xx. It honours GitHub rate limits and logs only exit codes.

## [0.3.0] - 2026-11-10

### Added

- Fleet maintenance. A nightly runner (`apps/api/cmd/fleetscan`, `.github/workflows/fleet-scan.yml`) reads every FacileStudio repository's lockfiles through a read-only GitHub App, runs douane, and delivers one gzip report to `POST /api/fleet/reports`. Sonde groups open findings by the upgrade that clears them and ranks them act, schedule or track; tracks version drift for muse, tronc and porte; and records each repository's scan, CI and Filet state. Setup is in `docs/FLEET.md`.
- A Maintenance page with three views: ranked upgrades, drift, and repositories with an editable role.
- Holds that keep a package on a version on purpose. Each needs a reason and an end date within a year.
- `SONDE_FLEET_TOKEN` turns report ingest on. `SONDE_FLEET_ADMINS` lists who may read fleet data.
- Migration `00006_fleet.sql`: new tables only.

### Security

- Fleet data needs `is_admin` and an email in `SONDE_FLEET_ADMINS`. `is_admin` alone is not enough, because erasing the last administrator promotes the lowest-numbered user. With both variables unset, ingest answers 404 and every read route answers 403.

### Changed

- The daily retention job also prunes resolved fleet findings and fleet runs after 90 days. Open findings are never pruned.

## [0.2.5] - 2026-11-07

### Changed

- build(client): bump muse to v0.16.1 (from v0.14.4). Secondary buttons use the `secondary` variant in place of the deprecated `outline`, and the remove button of a status page's monitor list uses `ghost-danger`. The icon generator accepts Iconify aliases, which the new muse icons need.
- `GET /version` comes from the `VERSION` file, which the release commit bumps (still `v` plus the number). The `APP_VERSION` build arg and the Dokploy env no longer take part, so a stale env value cannot make the server report an old release.

## [0.2.4] - 2026-11-07

### Changed

- build(client): bump muse to v0.14.4 (from v0.14.2). Switching the language in the user menu re-renders the whole UI at once, with no reload. A single 100% DonutChart slice draws as a full ring.

## [0.2.3] - 2026-11-07

### Security

- Registre tokens are sealed at rest. porte to v0.13.0: `porte_identities.access_token` and `refresh_token`
  are encrypted with AES-256-GCM under `PORTE_TOKEN_KEY`. Existing rows are sealed once, in the background,
  after boot, and the count is logged. An invalid `PORTE_TOKEN_KEY` stops the boot. With the key unset nothing changes.
- Losing `PORTE_TOKEN_KEY` makes the stored tokens unreadable: people sign in again, nothing else is lost.
  Do not roll back to a build older than porte v0.13.0 after the first reseal.

## [0.2.2] - 2026-11-07

### Changed

- `@facile/muse` v0.14.1 to v0.14.2: the user menu clamps itself inside the viewport at the right edge of a header.
- The user menu has a Subscription row that opens `https://sso.facile.studio/spaces`, where plans are managed.

## [0.2.1] - 2026-11-07

### Fixed

- The bundled Solar icon collection is regenerated for muse v0.14.1, so the user menu's glyphs (Help, external link, Language, Theme) are served locally instead of fetched from the icon API, which the CSP blocks.

## [0.2.0] - 2026-11-07

### Changed

- build(client): bump muse to v0.14.1. The sidebar user card and the mobile avatar open muse's `UserMenu` (Account, Settings, Language, Theme, Help, Sign out); the settings cog is gone. The language and theme picks are persisted (cookie and `sonde-theme`) and the theme is applied before hydration.
- Settings use muse's `SettingsLayout` in the canonical order: Profile, Preferences, Notifications, Bus, Advanced. Profile is read-only with "Manage on Facile"; Preferences (language and theme) replaces Appearance; the danger zone ends Advanced; sign-out left the settings and lives in the user menu.
- `/settings/appearance` redirects to `/settings/preferences`.
- The language picked in the menu now wins over the Registre profile locale on the next load.
- The uptime bar's empty cells use `fc-field` instead of the deprecated `fc-border`.

## [0.1.1] - 2026-11-07

### Changed

- build(api): bump porte to v0.12.1. The GDPR erasure no longer fails on a schema without `porte_magic_links`.
- build(client): bump muse to v0.13.0. Form controls are borderless fills, `Select` is muse's own combobox and badges stay on one line.
- The incident list on a monitor page marks each row with a filled dot instead of a left border.

## [0.1.0] - 2026-11-06

### Changed

- porte v0.5.2 to v0.12.0, porte/directory v0.3.0, tronc v0.16.0 to v0.17.1, @facile/muse v0.11.0 to v0.12.1.
- Sign-in goes through Registre only. The login page is muse `AuthScreen`, the landing page uses `LandingHeader`, `LandingFooter` and `SignInWithFacile`.
- An existing local account links by verified email at its first Facile sign-in, and only when it holds no other subject of the same issuer. An unverified claim never rewrites a stored address.
- `GET /api/auth/me` and `GET /api/users/me` return the user's `locale`.
- The UI is translated into fr, en, es and de. Dates, numbers and durations use Intl in the session locale.
- The production boot refuses to start without `OIDC_ISSUER`.
- tronc security headers are on: enforced anti-framing, report-only CSP with the client's inline script hashes.
- CI runs build, tests against Postgres, svelte-check and filet.

### Removed

- `POST /api/auth/register`, `POST /api/auth/login`, the local password kit, the register tab and the password form.
- `ALLOW_REGISTRATION` and `SSO_ONLY` (delete `SSO_ONLY` from the Dokploy env after deploy).

### Added

- Migration `00005_launch.sql`: `porte_login_codes.provider` and `.subject`, `users.locale`, `directory_state`.
- GDPR deletion: Registre erasures delete the person's monitors, webhooks, sessions and account. Erasing the only administrator promotes the oldest remaining account, so status pages stay manageable.
- `GET /version`, fed by the `APP_VERSION` build arg.
