#!/usr/bin/env sh
set -eu

data_dir=".local/postgres/data"

if ! command -v mise >/dev/null 2>&1; then
  echo "mise is required for db:down" >&2
  exit 1
fi

pg_root="$(mise where asdf:mise-plugins/mise-postgres)"
pg_bin_dir="$pg_root/bin"
pg_ctl_bin="$pg_bin_dir/pg_ctl"

if [ ! -x "$pg_ctl_bin" ]; then
  echo "mise-managed pg_ctl is not available" >&2
  exit 1
fi

if [ -d "$data_dir" ] && "$pg_ctl_bin" -D "$data_dir" status >/dev/null 2>&1; then
  "$pg_ctl_bin" -D "$data_dir" stop -m fast >/dev/null
  echo "stopped repo-local postgres"
else
  echo "repo-local postgres is not running"
fi
