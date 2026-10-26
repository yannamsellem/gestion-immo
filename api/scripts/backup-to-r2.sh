#!/usr/bin/env sh
set -eu
umask 077

: "${DATABASE_URL:?DATABASE_URL environment variable is required}"
: "${R2_ENDPOINT:?R2_ENDPOINT environment variable is required}"
: "${R2_ACCESS_KEY_ID:?R2_ACCESS_KEY_ID environment variable is required}"
: "${R2_SECRET_ACCESS_KEY:?R2_SECRET_ACCESS_KEY environment variable is required}"
: "${R2_BACKUP_BUCKET_NAME:=gestion-immo-backups}"

TIMESTAMP="$(date -u +%Y-%m-%dT%H-%M-%SZ)"
BACKUP_FILE="/tmp/gestion-immo-${TIMESTAMP}.sql.gz"
OBJECT_KEY="daily/gestion-immo-${TIMESTAMP}.sql.gz"

trap 'rm -f "${BACKUP_FILE}"' EXIT INT TERM

echo "[backup] creating compressed pg_dump archive ${OBJECT_KEY}..."
pg_dump --no-owner --no-acl -Z 9 -f "${BACKUP_FILE}" "${DATABASE_URL}"
[ -s "${BACKUP_FILE}" ] || { echo "[backup] error: backup file is empty" >&2; exit 1; }

AWS_ACCESS_KEY_ID="${R2_ACCESS_KEY_ID}" \
AWS_SECRET_ACCESS_KEY="${R2_SECRET_ACCESS_KEY}" \
AWS_DEFAULT_REGION="auto" \
aws s3 cp "${BACKUP_FILE}" "s3://${R2_BACKUP_BUCKET_NAME}/${OBJECT_KEY}" --endpoint-url "${R2_ENDPOINT}"

rm -f "${BACKUP_FILE}"
echo "[backup] upload completed: s3://${R2_BACKUP_BUCKET_NAME}/${OBJECT_KEY}"
