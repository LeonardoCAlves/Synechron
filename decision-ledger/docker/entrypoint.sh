#!/bin/sh
set -eu

DATABASE_URL="$MIGRATION_DATABASE_URL" ./node_modules/.bin/node-pg-migrate up -m migrations >&2
exec "$@"