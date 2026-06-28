#!/usr/bin/env bash
# Deprecated — use scripts/verify-docker.sh for final real-network validation.
exec "$(dirname "$0")/scripts/verify-docker.sh" "$@"
