# Authentication

Sonde authenticates through [porte](https://github.com/FacileStudio/porte) v0.3.x. One
session manager owns one sessions table; both kits issue through it, so a password login
and a federated one are the same row, the same cookie (`session`) and the same logout.

- **Local**: `POST /api/auth/register` + `POST /api/auth/login`, argon2id via porte/local.
  Gated by `ALLOW_REGISTRATION`; removed entirely when `SSO_ONLY=true` (not rejected —
  unregistered, so there is no endpoint to probe).
- **SSO**: `GET /api/auth/oidc` starts a PKCE flow against Authentik
  (`OIDC_ISSUER`); `GET /api/auth/oidc/callback` lands the user in the same session shape.
- **Config**: `GET /api/auth/config` returns `{sso_only, oidc_enabled,
  allow_registration}` so the login page knows what to draw.

## Login failure redirects

A failed OIDC login is a **302 redirect**, not a JSON error: porte sends the browser to
`FailureURL` (default `/login` on the success URL's origin) with `?error=<reason>`. The
client must read `Location`, not the status code, if it watches these endpoints, and must
render the `error` parameter **outside any SSO_ONLY gate** — under SSO_ONLY this redirect
is the only sign-in feedback a user will ever see.

## CSRF

Cookie-authenticated mutating requests require the `X-Facile-CSRF` header; bearer requests
are exempt by design. Verify client changes with the transport the browser actually uses.

## email_verified trap

Authentik's built-in `email` scope mapping hardcodes `email_verified=false`. Sonde does not
set `TrustEmailWithoutVerifiedClaim`, so a callback can never adopt an existing account by
address alone — silence is not verification. See
`~/.mycelium/memory/bugs/porte-adoption-null-oidc-subject.md` for the adoption history.
