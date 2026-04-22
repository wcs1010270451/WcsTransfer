#!/bin/sh
set -eu

BASE_URL="${BASE_URL:-}"
ADMIN_USER="${ADMIN_USER:-}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-}"

if [ -z "$BASE_URL" ]; then
  echo "BASE_URL is required"
  exit 1
fi

check_ok() {
  name="$1"
  url="$2"
  code="$(curl -ksS -o /tmp/wcstransfer-smoke.out -w "%{http_code}" "$url")"
  if [ "$code" -lt 200 ] || [ "$code" -ge 300 ]; then
    echo "smoke failed: ${name} http=${code}"
    cat /tmp/wcstransfer-smoke.out || true
    exit 1
  fi
  echo "ok: ${name}"
}

check_ok "healthz" "${BASE_URL}/healthz"
check_ok "version" "${BASE_URL}/version"

if [ -n "$ADMIN_USER" ] && [ -n "$ADMIN_PASSWORD" ]; then
  code="$(curl -ksS -u "${ADMIN_USER}:${ADMIN_PASSWORD}" -o /tmp/wcstransfer-console.out -w "%{http_code}" "${BASE_URL}/console/")"
  if [ "$code" -lt 200 ] || [ "$code" -ge 400 ]; then
    echo "smoke failed: console http=${code}"
    cat /tmp/wcstransfer-console.out || true
    exit 1
  fi
  echo "ok: console"
fi

echo "smoke ok"
