#!/usr/bin/env sh
set -eu

: "${DATABASE_URL:?DATABASE_URL environment variable is required}"
: "${R2_ENDPOINT:?R2_ENDPOINT environment variable is required}"
: "${R2_ACCESS_KEY_ID:?R2_ACCESS_KEY_ID environment variable is required}"
: "${R2_SECRET_ACCESS_KEY:?R2_SECRET_ACCESS_KEY environment variable is required}"
: "${R2_BACKUP_BUCKET_NAME:=gestion-immo-backups}"

TIMESTAMP="$(date -u +%Y-%m-%dT%H-%M-%SZ)"
BACKUP_FILE="/tmp/gestion-immo-${TIMESTAMP}.sql.gz"
OBJECT_KEY="daily/gestion-immo-${TIMESTAMP}.sql.gz"

echo "[backup] creating compressed pg_dump archive ${OBJECT_KEY}..."
pg_dump "${DATABASE_URL}" | gzip -9 > "${BACKUP_FILE}"

AWS_ACCESS_KEY_ID="${R2_ACCESS_KEY_ID}" \
AWS_SECRET_ACCESS_KEY="${R2_SECRET_ACCESS_KEY}" \
AWS_DEFAULT_REGION="auto" \
aws s3 cp "${BACKUP_FILE}" "s3://${R2_BACKUP_BUCKET_NAME}/${OBJECT_KEY}" --endpoint-url "${R2_ENDPOINT}"

rm -f "${BACKUP_FILE}"
echo "[backup] upload completed: s3://${R2_BACKUP_BUCKET_NAME}/${OBJECT_KEY}"
