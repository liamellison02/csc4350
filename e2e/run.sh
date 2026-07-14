#!/usr/bin/env bash
# opamp e2e: full stack up, round-trip test, always tear down.
set -euo pipefail
cd "$(dirname "$0")/.."

cleanup() {
  docker compose logs control-plane supervisor > /tmp/helmsman-e2e-logs.txt 2>&1 || true
  docker compose down -v
}
trap cleanup EXIT

docker compose up -d --build db api control-plane supervisor
API_URL="${API_URL:-http://localhost:8000}" uv run e2e/test_e2e.py
