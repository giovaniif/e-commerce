#!/bin/bash
set -e

CONNECT_URL="${KAFKA_CONNECT_URL:-http://kafka-connect:8083}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

wait_for_connect() {
  echo "Waiting for Kafka Connect to be ready..."
  until curl -s -o /dev/null -w "%{http_code}" "${CONNECT_URL}/connectors" | grep -q "200"; do
    sleep 2
  done
  echo "Kafka Connect is ready."
}

register_connector() {
  local name="$1"
  local file="$2"
  echo "Registering connector: ${name}"
  curl -s -X POST \
    -H "Content-Type: application/json" \
    --data "@${file}" \
    "${CONNECT_URL}/connectors" \
    | jq .
}

wait_for_connect

register_connector "order-outbox-connector"   "${SCRIPT_DIR}/order-connector.json"
register_connector "stock-outbox-connector"   "${SCRIPT_DIR}/stock-connector.json"
register_connector "payment-outbox-connector" "${SCRIPT_DIR}/payment-connector.json"

echo "All connectors registered."
