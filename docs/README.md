# Sonde — Documentation

| Page | What's in it |
|---|---|
| [Architecture](architecture.md) | Request flow, where the code lives, data model, auth |
| [Authentication](AUTH.md) | porte, Registre SSO, CSRF, the failure-redirect contract |
| [Retention](RETENTION.md) | The three data tiers, the prune job, the RGPD reasoning |
| [Configuration](configuration.md) | Every environment variable and default |
| [Development](development.md) | The dev database, the quality gate, the rename script |
| [Deployment](deployment.md) | The one-container shape, Traefik routing, verifying a deploy |
| [Adoption](ADOPTION.md) | How a suite app moves onto porte/kit, antenne-client v1, @facile/client and the shared gates |
| [API](api.md) | HTTP endpoints and payloads |
| [Plan](PLAN.md) | The v1 build plan and what is still open |

Sonde was forked from `GoSvelteBoilerplate`, so a few pages still describe template mechanics
alongside Sonde's own. `scripts/rename.sh` rewrites the docs, the root `README.md`,
`CLAUDE.md`, the app READMEs and `docker-compose.yml` in one pass.

Back to the [README](../README.md).
