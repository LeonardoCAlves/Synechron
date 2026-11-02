#!/usr/bin/env bash
set -euo pipefail

catalog_url="${CATALOG_URL:-http://127.0.0.1:8081}"
orders_url="${ORDERS_URL:-http://127.0.0.1:8082}"
response_file="$(mktemp)"
trap 'rm -f "$response_file"' EXIT

wait_ready() {
  local url="$1"
  local attempts=60
  until curl --fail --silent "$url/readyz" >/dev/null; do
    attempts=$((attempts - 1))
    if [ "$attempts" -le 0 ]; then
      echo "service did not become ready: $url" >&2
      return 1
    fi
    sleep 2
  done
}

wait_ready "$catalog_url"
wait_ready "$orders_url"

product_response="$(curl --fail --silent --show-error \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: compose-smoke-product-v1' \
  -d '{"name":"Smoke keyboard","description":"local smoke test","priceCents":1250,"currency":"CAD"}' \
  "$catalog_url/api/v1/products")"
product_id="$(printf '%s' "$product_response" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')"

attempts=30
while [ "$attempts" -gt 0 ]; do
  status="$(curl --silent --show-error -o "$response_file" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -H 'Idempotency-Key: compose-smoke-order-v1' \
    -d "{\"items\":[{\"productId\":\"$product_id\",\"quantity\":2}]}" \
    "$orders_url/api/v1/orders")"
  if [ "$status" = "201" ]; then
    python3 - "$response_file" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as response:
    order = json.load(response)
assert order["totalCents"] == 2500, order
assert order["items"][0]["productName"] == "Smoke keyboard", order
PY
    attempts=10
    until curl --fail --silent "$catalog_url/metrics" | grep -q '^order_platform_events_published_total [1-9]'; do
      attempts=$((attempts - 1))
      if [ "$attempts" -le 0 ]; then
        echo "catalog outbox publisher did not record the successful event" >&2
        exit 1
      fi
      sleep 1
    done
    echo "catalog-to-order Kafka smoke test passed"
    exit 0
  fi
  if [ "$status" != "503" ]; then
    cat "$response_file" >&2
    echo "order creation returned unexpected HTTP status $status" >&2
    exit 1
  fi
  attempts=$((attempts - 1))
  sleep 1
done

cat "$response_file" >&2
echo "catalog projection did not become available in time" >&2
exit 1
