#!/bin/sh
set -e
IFACE="${NETEM_IFACE:-eth0}"
if [ -n "$SCENARIO" ] && [ "$SCENARIO" != "clean" ]; then
  echo "[entrypoint] Applying netem scenario: $SCENARIO on $IFACE"
  SCENARIO="$SCENARIO" /netem.sh 10.10.0.10 "$IFACE"
fi
exec /client "$@"
