#!/bin/sh
set -eu

ENV_FILE="${1:-.env.prod}"
COMPOSE_FILE="${2:-docker-compose.prod.yml}"

if [ ! -f "$ENV_FILE" ]; then
  echo "missing env file: $ENV_FILE"
  exit 1
fi

if [ ! -f "$COMPOSE_FILE" ]; then
  echo "missing compose file: $COMPOSE_FILE"
  exit 1
fi

require_key() {
  key="$1"
  if ! grep -Eq "^${key}=" "$ENV_FILE"; then
    echo "missing key in ${ENV_FILE}: ${key}"
    exit 1
  fi
}

reject_pattern() {
  pattern="$1"
  message="$2"
  if grep -Eqi "$pattern" "$ENV_FILE"; then
    echo "$message"
    exit 1
  fi
}

require_key "DOMAIN"
require_key "PUBLIC_BASE_URL"
require_key "POSTGRES_PASSWORD"
require_key "AUTH_TOKEN_SECRET"
require_key "ADMIN_AUTH_TOKEN_SECRET"
require_key "ADMIN_BOOTSTRAP_PASSWORD"
require_key "ALERT_WEBHOOK_URL"

reject_pattern "change-me|replace_with|localhost|127\.0\.0\.1|example\.com" "env file still contains obvious placeholder or local values"

docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" config >/dev/null

echo "preflight ok"
