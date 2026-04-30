#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

pids=()

cleanup() {
  local exit_code=$?

  trap - EXIT INT TERM

  if ((${#pids[@]} > 0)); then
    kill "${pids[@]}" 2>/dev/null || true
    wait "${pids[@]}" 2>/dev/null || true
  fi

  exit "$exit_code"
}

start_service() {
  local service="$1"
  shift

  (
    cd "$REPO_ROOT/services/$service"
    env "$@" make run
  ) &

  pids+=("$!")
}

trap cleanup EXIT INT TERM

start_service unifi PORT=8081 AUTH_MODE=disabled UNIFI_API_URL=
start_service cluster PORT=8082 AUTH_MODE=disabled CLUSTER_API_URL=
start_service pfsense PORT=8083 AUTH_MODE=disabled PFSENSE_API_URL=
start_service polaris

printf '%s\n' 'Local platform is running.'
printf '%s\n' 'Polaris:   http://localhost:3000'
printf '%s\n' 'UniFi:     http://localhost:8081/api/v1/status'
printf '%s\n' 'Cluster:   http://localhost:8082/api/v1/status'
printf '%s\n' 'pfSense:   http://localhost:8083/api/v1/status'

wait -n
