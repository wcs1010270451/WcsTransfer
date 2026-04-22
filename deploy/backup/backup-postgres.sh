#!/bin/sh
set -eu

POSTGRES_HOST="${POSTGRES_HOST:-postgres}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_DB="${POSTGRES_DB:?set POSTGRES_DB}"
POSTGRES_USER="${POSTGRES_USER:?set POSTGRES_USER}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:?set POSTGRES_PASSWORD}"
BACKUP_DIR="${BACKUP_DIR:-/backups}"
BACKUP_INTERVAL_HOURS="${BACKUP_INTERVAL_HOURS:-24}"
BACKUP_RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-7}"
BACKUP_PREFIX="${BACKUP_PREFIX:-wcstransfer}"

mkdir -p "${BACKUP_DIR}"

run_backup() {
  timestamp="$(date -u +"%Y%m%dT%H%M%SZ")"
  filename="${BACKUP_DIR}/${BACKUP_PREFIX}_${POSTGRES_DB}_${timestamp}.sql.gz"
  checksum_file="${filename}.sha256"
  temp_file="${filename}.tmp"

  echo "[backup] starting backup at ${timestamp}"
  export PGPASSWORD="${POSTGRES_PASSWORD}"
  pg_dump \
    --host "${POSTGRES_HOST}" \
    --port "${POSTGRES_PORT}" \
    --username "${POSTGRES_USER}" \
    --dbname "${POSTGRES_DB}" \
    --clean \
    --if-exists \
    --no-owner \
    --no-privileges \
    | gzip -c > "${temp_file}"

  mv "${temp_file}" "${filename}"

  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${filename}" > "${checksum_file}"
  fi

  find "${BACKUP_DIR}" -type f -name "${BACKUP_PREFIX}_${POSTGRES_DB}_*.sql.gz" -mtime +"${BACKUP_RETENTION_DAYS}" -delete
  find "${BACKUP_DIR}" -type f -name "${BACKUP_PREFIX}_${POSTGRES_DB}_*.sql.gz.sha256" -mtime +"${BACKUP_RETENTION_DAYS}" -delete

  echo "[backup] completed: ${filename}"
}

run_backup

while true; do
  sleep "$((BACKUP_INTERVAL_HOURS * 3600))"
  run_backup
done
