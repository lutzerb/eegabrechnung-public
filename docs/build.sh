#!/usr/bin/env bash
# docs/build.sh — Build eegabrechnung-handbuch.pdf from Markdown chapters
set -euo pipefail

DOCS_DIR="$(cd "$(dirname "$0")" && pwd)"
OUT="$DOCS_DIR/eegabrechnung-handbuch.pdf"

# Check md-to-pdf is available
if ! command -v md-to-pdf &>/dev/null; then
  echo "Installing md-to-pdf..."
  npm install -g md-to-pdf
fi

# Chapter order
CHAPTERS=(
  "$DOCS_DIR/00-titelseite.md"
  "$DOCS_DIR/01-installation.md"
  "$DOCS_DIR/02-erste-schritte.md"
  "$DOCS_DIR/03-eeg-verwaltung.md"
  "$DOCS_DIR/04-mitglieder.md"
  "$DOCS_DIR/05-energiedaten.md"
  "$DOCS_DIR/06-tarifplaene.md"
  "$DOCS_DIR/07-abrechnung.md"
  "$DOCS_DIR/08-berichte.md"
  "$DOCS_DIR/09-buchhaltung.md"
  "$DOCS_DIR/10-sepa.md"
  "$DOCS_DIR/11-eda.md"
  "$DOCS_DIR/12-onboarding.md"
  "$DOCS_DIR/13-mitgliederportal.md"
  "$DOCS_DIR/14-mehrfachteilnahme.md"
  "$DOCS_DIR/15-benutzerverwaltung.md"
  "$DOCS_DIR/16-ea-buchhaltung.md"
  "$DOCS_DIR/A-datenbankschema.md"
  "$DOCS_DIR/B-api-referenz.md"
)

# Concatenate all chapters into a single temp file inside DOCS_DIR
# so that relative paths like screenshots/*.png resolve correctly
TMPFILE="$DOCS_DIR/_handbuch-build.md"
trap "rm -f $TMPFILE ${TMPFILE%.md}.pdf" EXIT

for chapter in "${CHAPTERS[@]}"; do
  if [[ -f "$chapter" ]]; then
    cat "$chapter" >> "$TMPFILE"
    echo -e "\n\n" >> "$TMPFILE"
  else
    echo "WARNING: Chapter not found: $chapter" >&2
  fi
done

# Regenerate swagger spec (requires Docker)
if command -v docker &>/dev/null; then
  echo "Regenerating Swagger spec..."
  docker run --rm \
    -v "$DOCS_DIR/../api:/app" \
    -w /app \
    golang:1.23-alpine \
    sh -c "go install github.com/swaggo/swag/cmd/swag@v1.16.4 2>/dev/null && swag init -g cmd/server/main.go -o docs 2>&1" || true
fi

# Regenerate swagger HTML
echo "Generating Swagger HTML..."
node "$DOCS_DIR/gen-swagger-html.mjs" || true

echo "Building PDF..."
md-to-pdf \
  --stylesheet "$DOCS_DIR/style.css" \
  --pdf-options '{"format":"A4","printBackground":true,"margin":{"top":"25mm","right":"20mm","bottom":"25mm","left":"25mm"}}' \
  --launch-options '{"executablePath":"/usr/bin/chromium-browser","args":["--no-sandbox","--disable-setuid-sandbox"]}' \
  "$TMPFILE"

# md-to-pdf writes to same dir as input with .pdf extension
GENERATED="${TMPFILE%.md}.pdf"
mv "$GENERATED" "$OUT"

echo "Done: $OUT"
