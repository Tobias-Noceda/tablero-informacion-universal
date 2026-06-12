#!/usr/bin/env bash
#
# Run the Go test suite with coverage and produce a report.
#
# Runs `go test` over the backend module with a cross-package coverage
# profile, prints a per-function summary plus the total, and writes an
# HTML report. Pass --open to launch the HTML in your browser.
#
# Usage:
#   ./coverage.sh
#   ./coverage.sh --open

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND="$SCRIPT_DIR/backend"

PROFILE="$BACKEND/coverage.out"
HTML="$BACKEND/coverage.html"

OPEN=0
if [[ "${1:-}" == "--open" ]]; then
    OPEN=1
fi

cd "$BACKEND"

echo "==> Running tests with coverage..."
go test ./common/... -covermode=atomic -coverprofile="$PROFILE"

echo
echo "==> Per-function coverage:"
go tool cover -func "$PROFILE"

go tool cover -html "$PROFILE" -o "$HTML"
echo
echo "==> HTML report: $HTML"

if [[ "$OPEN" -eq 1 ]]; then
    if command -v xdg-open >/dev/null 2>&1; then
        xdg-open "$HTML"
    elif command -v open >/dev/null 2>&1; then
        open "$HTML"
    else
        echo "No opener found; open the HTML manually."
    fi
fi
