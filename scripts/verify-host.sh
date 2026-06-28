#!/usr/bin/env bash
# Final validation on host LAN NIC (no loopback).
set -euo pipefail

CONNS="${1:-20}"
DUR="${2:-5s}"
BIND="${3:-0.0.0.0}"

NIC_IP="$(ip -4 route get 1.1.1.1 2>/dev/null | awk '{for(i=1;i<=NF;i++) if($i=="src") print $(i+1); exit}')"
if [ -z "$NIC_IP" ]; then
  NIC_IP="$(hostname -I | awk '{print $1}')"
fi
if [ -z "$NIC_IP" ] || [ "$NIC_IP" = "127.0.0.1" ]; then
  echo "No LAN IP; use scripts/verify-docker.sh"
  exit 1
fi

echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║  QCP verify-host — real UDP via NIC $NIC_IP"
echo "╚═══════════════════════════════════════════════════════════════╝"

cd "$BENCH"
go build -o qcp-bench .

./qcp-bench -mode server -bind "$BIND" &
SRV=$!
sleep 2
SERVER="${NIC_IP}:9000"
export QCP_BENCH_SERVER="$SERVER"

FAILED=0
for SCENARIO in lan wifi 4g 3g congested extreme; do
  echo ""
  echo "── scenario: $SCENARIO ──"
  export QCP_BENCH_SCENARIO="$SCENARIO"
  if ! ./qcp-bench -verify-net -server "$SERVER" -scenario "$SCENARIO" -duration "$DUR" -connections "$CONNS"; then
    FAILED=1
  fi
done

kill "$SRV" 2>/dev/null || true
[ "$FAILED" -eq 0 ] || exit 1
echo ""
echo "✓ HOST NIC VERIFY PASS"
