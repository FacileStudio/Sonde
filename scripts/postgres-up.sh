#!/usr/bin/env sh
set -eu

data_root=".local/postgres"
data_dir="$data_root/data"
log_file="$data_root/postgres.log"
host="127.0.0.1"
port="5433"
user="postgres"
database="api"

if ! command -v mise >/dev/null 2>&1; then
  echo "mise is required for db:up" >&2
  exit 1
fi

pg_root="$(mise where asdf:mise-plugins/mise-postgres)"
pg_bin_dir="$pg_root/bin"
initdb_bin="$pg_bin_dir/initdb"
pg_ctl_bin="$pg_bin_dir/pg_ctl"
pg_isready_bin="$pg_bin_dir/pg_isready"
psql_bin="$pg_bin_dir/psql"
createdb_bin="$pg_bin_dir/createdb"

if [ ! -x "$initdb_bin" ] || [ ! -x "$pg_ctl_bin" ] || [ ! -x "$pg_isready_bin" ] || [ ! -x "$psql_bin" ] || [ ! -x "$createdb_bin" ]; then
  echo "mise-managed postgres binaries are not available" >&2
  exit 1
fi

mkdir -p "$data_root"

if [ ! -s "$data_dir/PG_VERSION" ]; then
  echo "initializing postgres data directory at $data_dir"
  "$initdb_bin" -D "$data_dir" -U "$user" -A trust >/dev/null
fi

if "$pg_ctl_bin" -D "$data_dir" status >/dev/null 2>&1; then
  echo "postgres is already running on ${host}:${port}"
else
  echo "starting postgres on ${host}:${port}"
  "$pg_ctl_bin" -D "$data_dir" -l "$log_file" -o "-h ${host} -p ${port}" start >/dev/null
fi

echo "waiting for postgres to accept connections..."
until "$pg_isready_bin" -h "$host" -p "$port" -U "$user" -d postgres >/dev/null 2>&1; do
  sleep 1
done

if ! "$psql_bin" -h "$host" -p "$port" -U "$user" -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname = '${database}'" | grep -qx "1"; then
  "$createdb_bin" -h "$host" -p "$port" -U "$user" "$database"
fi

echo "postgres is ready: postgres://postgres:postgres@${host}:${port}/${database}?sslmode=disable"
