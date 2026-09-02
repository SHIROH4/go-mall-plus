#!/bin/bash
# Concurrent authenticated cart-write benchmark. This is a Redis write-path
# baseline; it deliberately does not create orders or consume product stock.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GATEWAY="${GATEWAY:-http://localhost:30088}"
THREADS="${THREADS:-4}"
CONNECTIONS="${CONNECTIONS:-8}"
DURATION="${DURATION:-20s}"
# User service limits usernames to 20 characters.
USERNAME="cb_$(date +%s)"
PASSWORD="test123456"

command -v curl >/dev/null 2>&1 || { echo "curl is required" >&2; exit 1; }
command -v jq >/dev/null 2>&1 || { echo "jq is required" >&2; exit 1; }
command -v wrk >/dev/null 2>&1 || { echo "wrk is required" >&2; exit 1; }

register_json=$(curl -fsS -X POST "$GATEWAY/api/user/register" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}")

if [ "$(printf '%s' "$register_json" | jq -r '.code // -1')" != "0" ]; then
  printf 'test user registration failed: %s\n' "$(printf '%s' "$register_json" | jq -r '.msg // "unknown error"')" >&2
  exit 1
fi

login_json=$(curl -fsS -X POST "$GATEWAY/api/user/login" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}")
auth_token=$(printf '%s' "$login_json" | jq -r '.data.accessToken // empty')

if [ -z "$auth_token" ]; then
  printf 'failed to obtain test token: %s\n' "$(printf '%s' "$login_json" | jq -r '.msg // "unknown error"')" >&2
  exit 1
fi

echo "Cart-write benchmark: threads=$THREADS connections=$CONNECTIONS duration=$DURATION"
echo "Target: $GATEWAY/api/cart/add"

AUTH_TOKEN="$auth_token" wrk \
  -t"$THREADS" \
  -c"$CONNECTIONS" \
  -d"$DURATION" \
  --latency \
  -s "$SCRIPT_DIR/wrk_cart_write.lua" \
  "$GATEWAY"
