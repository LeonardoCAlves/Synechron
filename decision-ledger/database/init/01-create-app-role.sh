#!/bin/sh
set -eu

psql --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" \
  --set=ON_ERROR_STOP=1 \
  --set=owner_role="$POSTGRES_USER" \
  --set=app_role="${APP_DATABASE_USER:-decision_ledger_app}" \
  --set=app_password="${APP_DATABASE_PASSWORD:-local_dev_only}" \
  --file=/docker-entrypoint-initdb.d/02-create-app-role.sql.template