#!/usr/bin/env bash
#
# Run the Go integration tests (`//go:build integration`).
#
# Brings up Mongo, Redis and the mock OAuth2 provider from docker-compose,
# points the tests at them through the environment (an ephemeral database
# name per run, dropped afterwards) and runs every tagged test in the module.
# `TestLiveDuende` also needs internet access; filter it out with -run if offline.
#
# Usage:
#   ./run_backend_integration.sh
#   ./run_backend_integration.sh -run TestEndToEnd
#   ./run_backend_integration.sh --keep      # leave the containers running afterwards

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

# Credentials come from .env, the same file docker compose reads.
set -a
# shellcheck disable=SC1091
source .env
set +a

MONGO_USER="${MONGO_INITDB_ROOT_USERNAME:?set MONGO_INITDB_ROOT_USERNAME in .env}"
MONGO_PASSWORD="${MONGO_INITDB_ROOT_PASSWORD:?set MONGO_INITDB_ROOT_PASSWORD in .env}"
IT_DATABASE="it_$(date +%s)"

export MONGODB_URI="mongodb://${MONGO_USER}:${MONGO_PASSWORD}@localhost:27017/?authSource=admin"
export MONGO_DATABASE="$IT_DATABASE"
export REDIS_URL="redis://localhost:6379/1"
export SECRETS_MASTER_KEYS="1:$(head -c 32 /dev/urandom | base64)"

echo "==> Starting Mongo, Redis and the mock OAuth2 provider..."
docker compose --profile integration up -d mongo redis mock-oauth2

cleanup() {
    echo "==> Dropping database $IT_DATABASE..."
    docker compose exec -T mongo mongosh --quiet -u "$MONGO_USER" -p "$MONGO_PASSWORD" --authenticationDatabase admin \
        --eval "db.getSiblingDB('$IT_DATABASE').dropDatabase()" >/dev/null || true
    if [[ "$KEEP" -eq 0 ]]; then
        echo "==> Stopping the mock OAuth2 provider..."
        docker compose --profile integration stop mock-oauth2 >/dev/null
    fi
}
trap cleanup EXIT

for _ in $(seq 1 30); do
    if curl -sf -o /dev/null "$MOCK_URL"; then
        break
    fi
    sleep 1
done
curl -sf -o /dev/null "$MOCK_URL" || { echo "Mock OAuth2 provider did not come up at $MOCK_URL"; exit 1; }

for _ in $(seq 1 30); do
    if docker compose exec -T mongo mongosh --quiet -u "$MONGO_USER" -p "$MONGO_PASSWORD" --authenticationDatabase admin --eval "db.runCommand({ping:1}).ok" >/dev/null 2>&1; then
        break
    fi
    sleep 1
done

cd "$BACKEND"

echo "==> Running integration tests against $IT_DATABASE..."
# Packages share one database, so they must not run concurrently.
go test -tags integration -count=1 -p 1 -v "${GO_ARGS[@]}" ./...
