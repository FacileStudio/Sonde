# Sonde — Development

Forking the template, running it locally, and the quality gate every fork inherits.

## Prerequisites

| Tool | Version | Why |
|---|---|---|
| mise | any | Task runner and toolchain pins. Activate it in your shell |
| Go | 1.25 | Pinned in `mise.toml`; `apps/api/go.mod` declares `go 1.25.7` |
| bun | 1.3.11 | Pinned in `mise.toml`. The client's only package manager |
| Docker | any | Preferred dev database provider. Optional — see the fallback below |

If monorepo tasks are not picked up automatically, `export MISE_EXPERIMENTAL=1` before running
anything.

## Forking

This is the first thing you do with a clone, before writing a line of code.

```sh
git clone git@github.com:FacileStudio/Sonde.git MyProject
cd MyProject
scripts/rename.sh MyProject
```

### Why the rename is not optional

A fresh fork inherits the Go module path
`github.com/FacileStudio/Sonde/apps/api`. Leaving it — or replacing it with a
bare `module api`, which is the other thing people did — means nothing in the suite can import
your packages, and no shared Go module can ever be introduced without touching every fork
first. Seven forks shipped that way before it was caught and fixed one repo at a time. The
script exists so that never has to happen again.

### What `scripts/rename.sh MyProject` does, in one pass

- Rewrites `module` in `apps/api/go.mod` to `github.com/FacileStudio/MyProject/apps/api`
- Rewrites every scoped Go import of the old module path, package root by package root
