#!/usr/bin/env bash
# Final validation: real UDP over Docker bridge + tc/netem (no loopback).
set -euo pipefail

CONNS="${1:-20}"
DUR="${2:-5s}"

echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║  QCP verify-docker — real network, all scenarios             ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
BENCH="$ROOT"

if [ ! -d "$ROOT/../qcp-lib-go" ]; then
  echo "Cloning qcp-lib-go for Docker build..."
  git clone --depth 1 https://github.com/neko233-com/qcp-lib-go.git "$ROOT/../qcp-lib-go"
fi

cd "$BENCH"
docker compose build
docker compose up -d server
sleep 2

FAILED=0
for SCENARIO in lan wifi 4g 3g congested extreme; do
  echo ""
  echo "── scenario: $SCENARIO ──"
  if ! docker compose run --rm \
    -e SCENARIO="$SCENARIO" \
    client \
    -verify-net \
    -server "10.10.0.10:9000" \
    -scenario "$SCENARIO" \
    -duration "$DUR" \
    -connections "$CONNS"; then
    FAILED=1
  fi
done

docker compose down
if [ "$FAILED" -ne 0 ]; then
  echo "VERIFY-DOCKER FAILED"
  exit 1
fi
echo ""
echo "✓ ALL DOCKER NETWORK SCENARIOS PASS"
