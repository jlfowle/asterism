#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SERVICES_DIR="$REPO_ROOT/services"
OUTPUT_PATH="$REPO_ROOT/services/polaris/public/microfrontends.json"
MODE="write"

usage() {
  cat <<'EOF'
Usage: scripts/generate-microfrontends-registry.sh [--write|--check]

Generate the Polaris microfrontend registry from the shared service inventory.
  --write  Regenerate services/polaris/public/microfrontends.json (default)
  --check  Fail if the committed registry is out of date
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --write)
      MODE="write"
      ;;
    --check)
      MODE="check"
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
  shift
done

inventory_json="$(cd "$SERVICES_DIR" && ./inventory.sh)"
registry_services='[]'

while IFS= read -r service_id; do
  module_path="$REPO_ROOT/services/$service_id/ui/module.json"
  if [[ ! -f "$module_path" ]]; then
    continue
  fi

  entry="$(
    jq -cn \
      --arg service "$service_id" \
      --slurpfile module_file "$module_path" '
        def join_path(base; path):
          if (base // "") == "" then
            path
          elif (path // "") == "" then
            base
          elif path | startswith("/") then
            base + path
          else
            base + "/" + path
          end;

        ($module_file[0]) as $manifest
        | {
            id: ($manifest.service // $service),
            displayName: ($manifest.dashboardCard.title // $manifest.displayName // $service),
            description: ($manifest.dashboardCard.description // ""),
            statusApi: join_path(($manifest.apiBasePath // ""); ($manifest.dashboardCard.statusEndpoint // "")),
            moduleManifest: join_path(($manifest.uiBasePath // ""); "module.json")
          }
      '
  )"

  registry_services="$(
    jq -cn \
      --argjson current "$registry_services" \
      --argjson entry "$entry" \
      '$current + [$entry]'
  )"
done < <(
  jq -r '
    .[]
    | select(.containerized == true and .has_kustomize_deployment == true)
    | .id
  ' <<<"$inventory_json"
)

rendered_registry="$(
  jq -n --argjson services "$registry_services" '{services: $services}'
)"

tmp_file="$(mktemp)"
trap 'rm -f "$tmp_file"' EXIT
printf '%s\n' "$rendered_registry" | jq '.' > "$tmp_file"

if [[ "$MODE" == "check" ]]; then
  if ! diff -u "$OUTPUT_PATH" "$tmp_file"; then
    echo "Polaris microfrontend registry is out of date. Run: scripts/generate-microfrontends-registry.sh --write" >&2
    exit 1
  fi
  echo "Polaris microfrontend registry is up to date."
  exit 0
fi

mv "$tmp_file" "$OUTPUT_PATH"
trap - EXIT
echo "Updated $OUTPUT_PATH"
