# Authentication

Sonde authenticates through [porte](https://github.com/FacileStudio/porte) v0.17.x, and only
through Registre. One session manager owns the sessions table (cookie `session`). There is no
local password, register or login route.

- **Account linking**: an account that predates the cutover links by verified email at its first
  Facile sign-in, and only when it holds no other subject of the same issuer.
- **Deletion**: `porte/directory` polls Registre for erased people; `UserStore.EraseAccount`
  deletes their monitors, webhooks, sessions and account. Status pages stay.
- **SSO**: `GET /api/auth/oidc` starts a PKCE flow against Registre (`OIDC_ISSUER`);
  `GET /api/auth/oidc/callback` lands the user in the same session shape.
- **Config**: `GET /api/auth/config` returns `{sso_only, oidc_enabled,
  allow_registration}` so the login page knows what to draw.

## The identity provider is Registre

The suite identity provider is **Registre**, a Go OIDC provider at
`https://sso.facile.studio`. It replaced Authentik, which no longer runs: there is no
container for it on la ruche, and `porte.facile.studio` returns 404 at the root and at the
discovery endpoint. Any config still pointing there is dead.

Do not confuse the two. **`porte` the Go auth kit** (`github.com/FacileStudio/porte`) is the
library Sonde imports and it is very much alive. **`porte.facile.studio`** was the Authentik
hostname and it is gone.

**Registre serves one issuer, at the root.**

```
OIDC_ISSUER=https://sso.facile.studio
```

Not a per-application path. Authentik minted a separate issuer per app slug
(`https://porte.facile.studio/application/o/sonde/`), which is why every app in the suite
changes its `OIDC_ISSUER` at cutover rather than just its hostname.

There is no alias that closes the gap. go-oidc compares the `issuer` field of the discovery
document against the configured URL byte for byte, so a trailing slash, a redirect, or a
CNAME will not do. Set the string exactly as written above.

## Sonde's client registration

| Setting | Value |
|---|---|
| `client_id` | `sonde` |
| Redirect URI | `https://sonde.facile.studio/api/auth/oidc/callback` |
| Back-channel logout | `https://sonde.facile.studio/api/auth/backchannel-logout` |

**The `/api` prefix is load-bearing and its absence is silent.** Every suite app mounts porte
inside `router.Route("/api", …)`. A redirect URI registered without the prefix does not 404,
because the SvelteKit catch-all claims everything the Go router has not, so the browser gets
200 with an HTML page and the login simply never completes. Nothing in the logs says the URI
was wrong. Read the path character by character when registering the client.

## Login failure redirects

A failed OIDC login is a **302 redirect**, not a JSON error: porte sends the browser to
`FailureURL` (default `/login` on the success URL's origin) with `?error=<reason>`. The
client must read `Location`, not the status code, if it watches these endpoints, and must
render the `error` parameter **outside any SSO_ONLY gate**. Under SSO_ONLY this redirect
is the only sign-in feedback a user will ever see.

## CSRF

Cookie-authenticated mutating requests require the `X-Facile-CSRF` header; bearer requests
are exempt by design. Verify client changes with the transport the browser actually uses.

## email_verified

Registre reports `email_verified` truthfully. Its accounts are minted by an operator, so they
are verified by construction and the claim says so.

Sonde still does not set `TrustEmailWithoutVerifiedClaim`, and should not. The flag tells
porte to adopt an existing account by email address when the provider sends no
`email_verified` claim at all. Registre sends one, so the flag would buy nothing and would
weaken the guarantee against a future provider that stays quiet.

The caution predates Registre and is worth keeping in mind when reading older suite
documents. Authentik's built-in `email` scope mapping hardcoded `email_verified: false`,
which broke porte's email fallback outright and is why those warnings exist across the suite.
Registre does not have that defect. See
`~/.mycelium/memory/bugs/porte-adoption-null-oidc-subject.md` for the adoption history.

## CLI sign-in

`OIDC_CLI_AUDIENCE` is empty by default. Setting it mounts porte's device-exchange route, so
the companion CLI can sign in against the same provider. Production value is `facile-cli`.
Leave it unset and the route does not exist.

Back to the [documentation index](README.md).
