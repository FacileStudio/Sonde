#!/usr/bin/env bash
#
# rename.sh - retarget a fresh fork of this template at its own identity.
#
#   scripts/rename.sh <NewRepoName>
#
# Rewrites the Go module path (go.mod plus every scoped import), the template
# name in the docs, and the dev-database identity (name and port) so that two
# forks can run `mise run db-up` side by side. Run once, on a clean git tree,
# immediately after cloning. Safe to re-run: it reports and exits if the fork
# has already been renamed.

set -euo pipefail

module_owner="${MODULE_OWNER:-FacileStudio}"
port_base=5433
port_span=200

fail() {
	printf 'rename: %s\n' "$1" >&2
	exit 1
}

usage() {
	printf 'usage: scripts/rename.sh <NewRepoName>\n' >&2
	printf '  e.g. scripts/rename.sh Sablier\n' >&2
}

escape_dots() {
	printf '%s' "$1" | sed 's/\./\\./g'
}

changed_files=""

rewrite() {
	local file="$1"
	shift
	local tmp="$file.rename-tmp"
	sed "$@" "$file" >"$tmp"
	if cmp -s "$file" "$tmp"; then
		rm -f "$tmp"
		return 1
	fi
	cat "$tmp" >"$file"
	rm -f "$tmp"
	case " $changed_files " in
	*" $file "*) ;;
	*) changed_files="$changed_files $file" ;;
	esac
	return 0
}

if [ "$#" -ne 1 ]; then
	usage
	exit 2
fi

new_name="$1"

if [ -z "$new_name" ]; then
	fail "repository name is empty"
fi
if ! printf '%s' "$new_name" | grep -Eq '^[A-Za-z][A-Za-z0-9._-]{1,63}$'; then
	fail "'$new_name' is not a plausible repository name (letters, digits, . _ - only; 2-64 chars; must start with a letter)"
fi
if printf '%s' "$new_name" | grep -q '\.\.'; then
	fail "'$new_name' is not a plausible repository name (contains '..')"
fi
case "$new_name" in
*[._-]) fail "'$new_name' is not a plausible repository name (trailing punctuation)" ;;
esac

repo_root="$(git rev-parse --show-toplevel 2>/dev/null)" || fail "not inside a git repository"
cd "$repo_root"

go_mod="apps/api/go.mod"
[ -f "$go_mod" ] || fail "$go_mod not found - run this from a checkout of the template"

old_module="$(sed -n 's/^module[[:space:]]\{1,\}\(.*\)$/\1/p' "$go_mod" | head -n 1)"
[ -n "$old_module" ] || fail "could not read the module path from $go_mod"

case "$old_module" in
*/apps/api) ;;
*) fail "module path '$old_module' does not end in /apps/api - rename this fork by hand" ;;
esac

old_repo="$(basename "$(dirname "$(dirname "$old_module")")")"
old_slug="$(printf '%s' "$old_repo" | tr '[:upper:]' '[:lower:]')"

new_module="github.com/$module_owner/$new_name/apps/api"
new_slug="$(printf '%s' "$new_name" | tr '[:upper:]' '[:lower:]' | tr -c 'a-z0-9' '_' | tr -s '_' | sed 's/^_//; s/_$//')"
[ -n "$new_slug" ] || fail "'$new_name' does not reduce to a usable database name"

port_offset="$(printf '%s' "$new_slug" | cksum | awk '{print $1}')"
db_port=$((port_base + port_offset % port_span))

pg_up="scripts/postgres-up.sh"
old_db_port="$port_base"
old_db_name="api"
if [ -f "$pg_up" ]; then
	found_port="$(sed -n 's/^port="\${POSTGRES_PORT:-\([0-9]\{1,\}\)}"$/\1/p' "$pg_up" | head -n 1)"
	found_name="$(sed -n 's/^database="\${POSTGRES_DB:-\([A-Za-z0-9_]\{1,\}\)}"$/\1/p' "$pg_up" | head -n 1)"
	[ -n "$found_port" ] && old_db_port="$found_port"
	[ -n "$found_name" ] && old_db_name="$found_name"
fi

token_file="apps/client/src/lib/backend.ts"
old_token_key="app.token"
if [ -f "$token_file" ]; then
	found_token_key="$(sed -n "s/^export const tokenStorageKey = '\([A-Za-z0-9._-]\{1,\}\)';\$/\1/p" "$token_file" | head -n 1)"
	[ -n "$found_token_key" ] && old_token_key="$found_token_key"
fi
new_token_key="$new_slug.token"

if [ "$old_module" = "$new_module" ]; then
	printf 'rename: already renamed to %s - nothing to do\n' "$new_name"
	exit 0
fi
if [ "$old_repo" = "$new_name" ]; then
	fail "module path is '$old_module' but the repo is already named '$new_name' - fix $go_mod by hand"
fi

if [ -n "$(git status --porcelain)" ]; then
	fail "working tree is dirty - commit or stash first so the rename is reviewable"
fi

pkg_dirs="$(find apps/api -mindepth 2 -name '*.go' -not -path '*/.*' -print |
	sed 's|^apps/api/||' | cut -d/ -f1 | sort -u)"
[ -n "$pkg_dirs" ] || fail "found no Go package directories under apps/api"

old_module_re="$(escape_dots "$old_module")"

import_exprs=()
import_exprs+=(-e "s|\"$old_module_re\"|\"$new_module\"|g")
for dir in $pkg_dirs; do
	import_exprs+=(-e "s|\"$old_module_re/$dir\"|\"$new_module/$dir\"|g")
	import_exprs+=(-e "s|\"$old_module_re/$dir/|\"$new_module/$dir/|g")
done

rewrite "$go_mod" -e "s|^module[[:space:]]\{1,\}$old_module_re\$|module $new_module|" ||
	fail "failed to rewrite the module path in $go_mod"

go_files_changed=0
while IFS= read -r file; do
	[ -n "$file" ] || continue
	if rewrite "$file" "${import_exprs[@]}"; then
		go_files_changed=$((go_files_changed + 1))
	fi
done <<EOF
$(grep -rl "$old_module" --include='*.go' . || true)
EOF

doc_files=""
for candidate in README.md CLAUDE.md apps/api/README.md apps/client/README.md docs/*.md; do
	[ -f "$candidate" ] && doc_files="$doc_files $candidate"
done

# docker-compose.yml carries the public hostname and the Traefik router names,
# so it has to follow the slug or a fork deploys itself onto app.facile.studio.
[ -f docker-compose.yml ] && doc_files="$doc_files docker-compose.yml"

# app.html carries the tab title, the description and the absolute og:image and
# og:url. Those have to be absolute for social scrapers, which means a fork that
# skips this ships the template's name and links every preview at the template's
# hostname. Same substitution as the docs: repo name and slug.
[ -f apps/client/src/app.html ] && doc_files="$doc_files apps/client/src/app.html"

for file in $doc_files; do
	rewrite "$file" \
		-e '/<!-- fork:begin -->/,/<!-- fork:end -->/d' \
		-e "s|$(escape_dots "$old_repo")|$new_name|g" \
		-e "s|$(escape_dots "$old_slug")|$new_slug|g" || true
done

db_files=""
for candidate in scripts/postgres-up.sh scripts/postgres-down.sh apps/api/mise.toml apps/api/.env.example apps/api/.env README.md CLAUDE.md docs/*.md; do
	[ -f "$candidate" ] && db_files="$db_files $candidate"
done

for file in $db_files; do
	rewrite "$file" \
		-e "s|:$old_db_port/$old_db_name|:$db_port/$new_slug|g" \
		-e "s|POSTGRES_DB:-$old_db_name}|POSTGRES_DB:-$new_slug}|g" \
		-e "s|POSTGRES_PORT:-$old_db_port}|POSTGRES_PORT:-$db_port}|g" \
		-e "s|127\\.0\\.0\\.1:$old_db_port|127.0.0.1:$db_port|g" \
		-e "s|port $old_db_port|port $db_port|g" || true
done

if [ -f "$token_file" ]; then
	rewrite "$token_file" \
		-e "s|^export const tokenStorageKey = '$(escape_dots "$old_token_key")';\$|export const tokenStorageKey = '$new_token_key';|" || true
fi

# The Go rewrite above only touches import paths, so the API reference at /docs
# would keep the template's name in its title and its browser tab. It is a
# single-line const in main.go precisely so this sed can find it.
ref_file="apps/api/main.go"
if [ -f "$ref_file" ]; then
	rewrite "$ref_file" \
		-e "s|^const apiTitle = \"$(escape_dots "$old_repo") API\"\$|const apiTitle = \"$new_name API\"|" || true
fi

printf '\nRenamed %s -> %s\n\n' "$old_repo" "$new_name"
printf '  go module   %s\n' "$new_module"
printf '  go files    %d rewritten (package roots: %s)\n' "$go_files_changed" "$(printf '%s' "$pkg_dirs" | tr '\n' ' ')"
printf '  dev db      %s on 127.0.0.1:%d\n' "$new_slug" "$db_port"
printf '  token key   %s\n' "$new_token_key"
printf '  overrides   POSTGRES_DB / POSTGRES_PORT\n'
printf '\nFiles changed:\n'
for file in $changed_files; do
	printf '  %s\n' "$file"
done
printf '\nNext:\n'
printf '  git remote set-url origin git@github.com:%s/%s.git\n' "$module_owner" "$new_name"
printf '  mise //apps/api:tidy\n'
printf '  mise run db-up && mise run dev\n'
printf '  review CLAUDE.md and the READMEs for prose that still describes the template\n'
printf '  set apiDescription in apps/api/main.go - only the title is rewritten\n'
printf '  delete the demo domain (modules/events, modules/tickets, internal/ticketcode, cmd/seed, schemas/event.go, schemas/ticket.go)\n'
