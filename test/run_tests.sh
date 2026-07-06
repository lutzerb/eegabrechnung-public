#!/usr/bin/env bash
# Run the eegabrechnung integration test suite.
#
# Usage:
#   ./run_tests.sh              # Run all tests (skips EDA mail tests if worker not running)
#   ./run_tests.sh --mail       # Also start the test profile (mailpit + eda-worker-test)
#   ./run_tests.sh -k billing   # Run only tests matching "billing"
#
# Prerequisites:
#   cd /mnt/HC_Volume_103451728/eegabrechnung/test
#   python3 -m venv .venv && source .venv/bin/activate
#   pip install -r requirements.txt

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$SCRIPT_DIR"

# ── Optional: start test profile ─────────────────────────────────────────────
START_TEST_PROFILE=false
EXTRA_PYTEST_ARGS=()

for arg in "$@"; do
    case "$arg" in
        --mail)
            START_TEST_PROFILE=true
            ;;
        *)
            EXTRA_PYTEST_ARGS+=("$arg")
            ;;
    esac
done

if $START_TEST_PROFILE; then
    echo "▶ Starting test docker compose profile (mailpit + eda-worker-test)..."
    docker compose --project-dir "$PROJECT_DIR" --profile test up -d
    echo "⏳ Waiting for services to be ready..."
    sleep 5
fi

# ── Check API is reachable ────────────────────────────────────────────────────
echo "▶ Checking API health..."
API_URL="${API_URL:-http://localhost:8101}"
if ! curl -sf "${API_URL}/api/v1/health" > /dev/null; then
    echo "✗ API not reachable at ${API_URL}. Is the stack running?"
    echo "  docker compose up -d"
    exit 1
fi
echo "✓ API is healthy"

# ── Run tests ─────────────────────────────────────────────────────────────────
echo "▶ Running pytest..."
python3 -m pytest "${EXTRA_PYTEST_ARGS[@]}" -v "$@" 2>&1 | grep -v "^$" || true

# Exit with pytest's exit code.
python3 -m pytest "${EXTRA_PYTEST_ARGS[@]}" -v --tb=short
