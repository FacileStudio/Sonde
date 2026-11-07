# Changelog

## [Unreleased]

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
