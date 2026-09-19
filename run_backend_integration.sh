#!/usr/bin/env bash
#
# Run the Go integration tests (`//go:build integration`).
#
# Starts the mock OAuth2 provider from docker-compose (profile `integration`,
# listening on localhost:8899), waits for it, and runs the tagged tests. The
# Duende test also needs internet access; filter it out with -run if offline.
#
# Usage:
#   ./run_backend_integration.sh
#   ./run_backend_integration.sh -run TestLiveAuthorizationCode
#   ./run_backend_integration.sh --keep      # leave the mock running afterwards

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND="$SCRIPT_DIR/backend"

MOCK_URL="http://localhost:8899/default/.well-known/openid-configuration"

KEEP=0
GO_ARGS=()
for arg in "$@"; do
    case "$arg" in
        --keep) KEEP=1 ;;
        *) GO_ARGS+=("$arg") ;;
    esac
done

cd "$SCRIPT_DIR"

echo "==> Starting mock OAuth2 provider..."
docker compose --profile integration up -d mock-oauth2

if [[ "$KEEP" -eq 0 ]]; then
    trap 'echo "==> Stopping mock OAuth2 provider..."; docker compose --profile integration stop mock-oauth2 >/dev/null' EXIT
fi

for _ in $(seq 1 30); do
    if curl -sf -o /dev/null "$MOCK_URL"; then
        break
    fi
    sleep 1
done
curl -sf -o /dev/null "$MOCK_URL" || { echo "Mock OAuth2 provider did not come up at $MOCK_URL"; exit 1; }

cd "$BACKEND"

echo "==> Running integration tests..."
go test -tags integration -v "${GO_ARGS[@]}" ./common/services/secrets/
