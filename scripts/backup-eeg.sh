#!/bin/bash
set -euo pipefail

BORG_REPO="/mnt/HC_Volume_103451728/backups/borg-eeg"
PG_CONTAINER="eegabrechnung-eegabrechnung-postgres-1"
STAGING=$(mktemp -d)
# Cleanup via Docker weil Alpine-Container Dateien als root anlegt
trap "docker run --rm -v $STAGING:/staging alpine rm -rf /staging/invoices /staging/documents 2>/dev/null; rm -rf $STAGING" EXIT

# 1. pg_dump (custom format = intern komprimiert)
docker exec "$PG_CONTAINER" \
  pg_dump -U eegabrechnung -Fc eegabrechnung \
  > "$STAGING/db.dump"

# 2. Dateien aus Docker Volumes (mit aktuellem User, damit cleanup klappt)
docker run --rm \
  --user "$(id -u):$(id -g)" \
  -v eegabrechnung_eegabrechnung_invoice_data:/invoices:ro \
  -v eegabrechnung_eegabrechnung_document_data:/documents:ro \
  -v "$STAGING:/out" \
  alpine sh -c "cp -r /invoices /out/ && cp -r /documents /out/"

# 3. Borg create (lz4 = schnell, gute Kompression)
borg create \
  --compression lz4 \
  --stats \
  "$BORG_REPO::eeg-{now:%Y-%m-%dT%H%M}" \
  "$STAGING" \
  2>&1

# 4. Retention: 7 Tage täglich
borg prune \
  --keep-daily=7 \
  --stats \
  "$BORG_REPO" \
  2>&1

echo "Backup abgeschlossen: $(date)"
