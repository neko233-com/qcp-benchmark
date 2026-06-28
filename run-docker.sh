#!/bin/bash
# Run benchmarks with Docker network simulation
# Usage: ./run-docker.sh [scenario] [connections] [duration]

set -e

SCENARIO=${1:-lan}
CONNS=${2:-100}
DUR=${3:-10s}

echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║         QCP Docker Benchmark Runner                        ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
echo ""
echo "Scenario:   $SCENARIO"
echo "Connection: $CONNS"
echo "Duration:   $DUR"
echo ""

# Build images
echo "Building images..."
docker-compose build

# Start server
echo "Starting server..."
docker-compose up -d server
sleep 2

# Run client with network simulation
echo "Running client with $SCENARIO network..."
docker-compose run --rm \
  -e SCENARIO=$SCENARIO \
  client \
  -mode client \
  -server 10.10.0.10:9000 \
  -protocol all \
  -duration $DUR \
  -connections $CONNS

# Cleanup
echo ""
echo "Cleaning up..."
docker-compose down

echo ""
echo "Done! Results saved in client container at /results.json"
