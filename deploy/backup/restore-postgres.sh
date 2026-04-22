#!/bin/sh
set -eu

if [ "$#" -lt 1 ]; then
  echo "usage: /bin/sh restore-postgres.sh <backup-file.sql.gz|backup-file.sql>" >&2
  exit 1
fi

BACKUP_FILE="$1"
POSTGRES_HOST="${POSTGRES_HOST:-postgres}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_DB="${POSTGRES_DB:?set POSTGRES_DB}"
POSTGRES_USER="${POSTGRES_USER:?set POSTGRES_USER}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:?set POSTGRES_PASSWORD}"

if [ ! -f "${BACKUP_FILE}" ]; then
  echo "backup file not found: ${BACKUP_FILE}" >&2
  exit 1
fi

export PGPASSWORD="${POSTGRES_PASSWORD}"

echo "[restore] restoring ${BACKUP_FILE} into ${POSTGRES_DB} on ${POSTGRES_HOST}:${POSTGRES_PORT}"

case "${BACKUP_FILE}" in
  *.sql.gz)
    gzip -dc "${BACKUP_FILE}" | psql \
      --host "${POSTGRES_HOST}" \
      --port "${POSTGRES_PORT}" \
      --username "${POSTGRES_USER}" \
      --dbname "${POSTGRES_DB}"
    ;;
  *.sql)
    psql \
      --host "${POSTGRES_HOST}" \
      --port "${POSTGRES_PORT}" \
      --username "${POSTGRES_USER}" \
      --dbname "${POSTGRES_DB}" \
      --file "${BACKUP_FILE}"
    ;;
  *)
    echo "unsupported backup file extension: ${BACKUP_FILE}" >&2
    exit 1
    ;;
esac

echo "[restore] completed"
