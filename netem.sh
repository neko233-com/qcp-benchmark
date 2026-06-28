#!/bin/bash
# Network simulation scenarios using tc/netem
# Run inside the client container

set -e

SERVER_IP=${1:-10.10.0.10}
INTERFACE=${2:-eth0}

echo "Configuring network on $INTERFACE to $SERVER_IP"

# Clear existing rules
tc qdisc del dev $INTERFACE root 2>/dev/null || true

case "${SCENARIO:-clean}" in
  clean)
    echo "Clean network (no delay/loss)"
    ;;

  lan)
    echo "LAN: 1ms delay, 0.1% loss"
    tc qdisc add dev $INTERFACE root netem delay 1ms loss 0.1%
    ;;

  wifi)
    echo "WiFi: 10ms delay, 2% loss, 1% duplicate"
    tc qdisc add dev $INTERFACE root netem delay 10ms loss 2% duplicate 1%
    ;;

  4g)
    echo "4G: 30ms delay, 1% loss"
    tc qdisc add dev $INTERFACE root netem delay 30ms loss 1%
    ;;

  3g)
    echo "3G: 100ms delay, 3% loss"
    tc qdisc add dev $INTERFACE root netem delay 100ms loss 3%
    ;;

  congested)
    echo "Congested: 50ms delay, 5% loss, 1% corruption"
    tc qdisc add dev $INTERFACE root netem delay 50ms loss 5% corrupt 1%
    ;;

  extreme)
    echo "Extreme: 100ms delay, 10% loss, 2% corruption"
    tc qdisc add dev $INTERFACE root netem delay 100ms loss 10% corrupt 2%
    ;;

 *)
    echo "Unknown scenario: $SCENARIO"
    echo "Available: clean, lan, wifi, 4g, 3g, congested, extreme"
    exit 1
    ;;
esac

echo "Network configured. Scenario: $SCENARIO"
