# GoSvelteBoilerplate — Documentation

| Page | What's in it |
|---|---|
| [Architecture](architecture.md) | Request flow, the module pattern, data model, auth |
| [Configuration](configuration.md) | Every environment variable and default |
| [Development](development.md) | Forking, the rename script, the dev database, the quality gate |
| [Deployment](deployment.md) | The one-container shape, Traefik routing, verifying a deploy |
| [API](api.md) | HTTP endpoints and payloads |

These pages describe the template. `scripts/rename.sh` rewrites them along with the root
`README.md`, `CLAUDE.md`, the app READMEs and `docker-compose.yml`, so a fresh fork's docs
carry its own name and dev-database port. Prose that describes template behaviour rather than
the fork's still needs a read-through by hand.

Back to the [README](../README.md).
