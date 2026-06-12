#!/usr/bin/env bash
#
# Run the Go test suite with coverage and produce a report.
#
# Runs `go test` over the backend module (excluding test doubles and the live
# driver wrappers), prints the per-function breakdown followed by a per-package
# + total summary, and writes an HTML report.
#
# Usage:
#   ./run_backend_coverage.sh
#   ./run_backend_coverage.sh --open

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND="$SCRIPT_DIR/backend"

PROFILE="$BACKEND/coverage.out"
HTML="$BACKEND/coverage.html"

EXCLUDE='ports/(mongo|redis)|/mocks'

OPEN=0
for arg in "$@"; do
    case "$arg" in
        --open) OPEN=1 ;;
    esac
done

cd "$BACKEND"

PKGS=$(go list ./common/... | grep -vE "$EXCLUDE")

echo "==> Running tests with coverage..."
# -covermode=atomic records per-block hit counts (goroutine-safe; the cache
# write in ExecutePostIt runs in a goroutine), surfacing weakly-hit branches.
TEST_OUT=$(go test $PKGS -covermode=atomic -coverprofile="$PROFILE" -cover)
echo "$TEST_OUT"

FUNC_OUT=$(go tool cover -func "$PROFILE")

echo
echo "==> Per-function coverage:"
echo "$FUNC_OUT"

echo
echo "==> Coverage summary"

# Statement coverage per package (from `go test`). Portable awk -> "pkg pct".
declare -A STMT
while read -r pkg pct; do
    [[ -n "$pkg" ]] && STMT["$pkg"]="$pct"
done < <(echo "$TEST_OUT" | awk '
    /coverage:/ {
        pct = ""
        for (i = 1; i <= NF; i++) if ($i == "coverage:") { pct = $(i + 1); sub(/%/, "", pct) }
        pkg = $2; sub(/.*\/common\//, "", pkg)
        print pkg, pct
    }')

# Function coverage per package: a function counts as covered if any of its
# statements ran
declare -A FUNCP
while read -r pkg pct; do
    [[ -n "$pkg" ]] && FUNCP["$pkg"]="$pct"
done < <(echo "$FUNC_OUT" | awk '
    !/^total:/ {
        pct = $NF; sub(/%/, "", pct)
        pkg = $1; sub(/.*\/common\//, "", pkg); sub(/\/[^\/]+\.go:.*/, "", pkg)
        tot[pkg]++; if (pct + 0 > 0) cov[pkg]++
        gtot++; if (pct + 0 > 0) gcov++
    }
    END {
        for (p in tot) printf "%s %.1f\n", p, 100 * cov[p] / tot[p]
        if (gtot > 0) printf "TOTAL %.1f\n", 100 * gcov / gtot
    }')

TOTAL_STMT=$(echo "$FUNC_OUT" | awk '/^total:/ { gsub(/%/, "", $NF); print $NF }')

printf "    %-22s %7s %7s\n" "package" "stmt" "func"
for pkg in $(printf '%s\n' "${!STMT[@]}" | sort); do
    printf "    %-22s %6s%% %6s%%\n" "$pkg" "${STMT[$pkg]}" "${FUNCP[$pkg]:-0.0}"
done
printf "    %-22s %6s%% %6s%%\n" "TOTAL" "$TOTAL_STMT" "${FUNCP[TOTAL]:-0.0}"

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
