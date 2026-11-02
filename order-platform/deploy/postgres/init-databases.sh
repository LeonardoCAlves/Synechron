#!/bin/bash
set -euo pipefail

psql \
  --set=ON_ERROR_STOP=1 \
  --username "$POSTGRES_USER" \
  --dbname "$POSTGRES_DB" \
  --set=catalog_password="$CATALOG_DB_PASSWORD" \
  --set=orders_password="$ORDERS_DB_PASSWORD" <<'SQL'
CREATE ROLE catalog_user LOGIN PASSWORD :'catalog_password';
CREATE DATABASE catalog_service OWNER catalog_user;
CREATE ROLE orders_user LOGIN PASSWORD :'orders_password';
CREATE DATABASE orders_service OWNER orders_user;
SQL
