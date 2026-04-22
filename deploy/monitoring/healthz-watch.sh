#!/bin/sh
set -eu

HEALTHCHECK_URL="${HEALTHCHECK_URL:-}"
HEALTHCHECK_INTERVAL="${HEALTHCHECK_INTERVAL:-30}"
ALERT_WEBHOOK_URL="${ALERT_WEBHOOK_URL:-}"
ALERT_WEBHOOK_PROVIDER="${ALERT_WEBHOOK_PROVIDER:-generic}"
APP_NAME="${APP_NAME:-wcstransfer-gateway}"

if [ -z "$HEALTHCHECK_URL" ]; then
  echo "HEALTHCHECK_URL is required"
  exit 1
fi

STATE_FILE="/tmp/healthz-watch.state"

send_alert() {
  status="$1"
  details="$2"

  if [ -z "$ALERT_WEBHOOK_URL" ]; then
    return 0
  fi

  message="[WcsTransfer] healthz异常 service=${APP_NAME} status=${status} url=${HEALTHCHECK_URL} details=${details}"
  if [ "$status" = "recovered" ]; then
    message="[WcsTransfer] healthz恢复 service=${APP_NAME} url=${HEALTHCHECK_URL}"
  fi

  case "$ALERT_WEBHOOK_PROVIDER" in
    wecom)
      payload=$(printf '{"msgtype":"text","text":{"content":"%s"}}' "$message")
      ;;
    feishu)
      payload=$(printf '{"msg_type":"text","content":{"text":"%s"}}' "$message")
      ;;
    *)
      payload=$(printf '{"event":"healthz_watch","level":"%s","source":"wcstransfer.healthz","message":"%s","data":{"service":"%s","url":"%s","details":"%s"}}' "$status" "$message" "$APP_NAME" "$HEALTHCHECK_URL" "$details")
      ;;
  esac

  curl -fsS -X POST "$ALERT_WEBHOOK_URL" \
    -H "Content-Type: application/json" \
    -d "$payload" >/dev/null || true
}

is_alerted() {
  [ -f "$STATE_FILE" ] && [ "$(cat "$STATE_FILE" 2>/dev/null || true)" = "down" ]
}

mark_down() {
  echo "down" >"$STATE_FILE"
}

clear_down() {
  rm -f "$STATE_FILE"
}

while true; do
  tmp_body="/tmp/healthz-body.$$"
  http_code="$(curl -sS -o "$tmp_body" -w "%{http_code}" "$HEALTHCHECK_URL" || echo 000)"
  body="$(cat "$tmp_body" 2>/dev/null || true)"
  rm -f "$tmp_body"

  if [ "$http_code" = "200" ]; then
    if is_alerted; then
      send_alert "recovered" ""
      clear_down
    fi
  else
    if ! is_alerted; then
      details="http_code=${http_code}"
      if [ -n "$body" ]; then
        details="${details} body=${body}"
      fi
      send_alert "down" "$details"
      mark_down
    fi
  fi

  sleep "$HEALTHCHECK_INTERVAL"
done
