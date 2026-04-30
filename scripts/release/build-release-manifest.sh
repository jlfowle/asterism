#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: build-release-manifest.sh --release-dir DIR --output FILE --version VERSION --commit SHA --expected-services-json JSON

Builds release-manifest.json from release image metadata and fails if the
manifest would be empty or incomplete.
EOF
}

RELEASE_DIR=""
OUTPUT=""
VERSION=""
COMMIT=""
EXPECTED_SERVICES_JSON="${EXPECTED_SERVICES_JSON:-}"
SOURCE_CI_RUN_ID="${SOURCE_CI_RUN_ID:-}"
SOURCE_CI_RUN_URL="${SOURCE_CI_RUN_URL:-}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --release-dir)
      RELEASE_DIR="$2"
      shift 2
      ;;
    --output)
      OUTPUT="$2"
      shift 2
      ;;
    --version)
      VERSION="$2"
      shift 2
      ;;
    --commit)
      COMMIT="$2"
      shift 2
      ;;
    --expected-services-json)
      EXPECTED_SERVICES_JSON="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$RELEASE_DIR" || -z "$OUTPUT" || -z "$VERSION" || -z "$COMMIT" || -z "$EXPECTED_SERVICES_JSON" ]]; then
  usage >&2
  exit 2
fi

if [[ ! -d "$RELEASE_DIR" ]]; then
  echo "Release metadata directory does not exist: $RELEASE_DIR" >&2
  exit 1
fi

if ! jq -e 'type == "array" and length > 0 and all(.[]; (.id | type == "string") and (.id | length > 0))' <<< "$EXPECTED_SERVICES_JSON" > /dev/null; then
  echo "Expected services JSON must be a non-empty array of service objects with id fields." >&2
  exit 1
fi

mapfile -t expected_services < <(jq -r '.[].id' <<< "$EXPECTED_SERVICES_JSON" | sort)
metadata_files=()

for service in "${expected_services[@]}"; do
  metadata="$RELEASE_DIR/image-metadata-${service}.json"
  if [[ ! -s "$metadata" ]]; then
    echo "Missing release image metadata for ${service}: $metadata" >&2
    exit 1
  fi
  jq -e \
    --arg service "$service" \
    --arg version "$VERSION" \
    '.service == $service and .version == $version and (.image | type == "string" and length > 0) and (.digest | test("^sha256:[0-9a-fA-F]{64}$"))' \
    "$metadata" > /dev/null
  metadata_files+=("$metadata")
done

services_tmp="$(mktemp)"
jq -s 'sort_by(.service)' "${metadata_files[@]}" > "$services_tmp"

service_count="$(jq 'length' "$services_tmp")"
if [[ "$service_count" -eq 0 ]]; then
  echo "Refusing to build a release manifest with zero services." >&2
  exit 1
fi

if [[ "$service_count" -ne "${#expected_services[@]}" ]]; then
  echo "Release manifest service count does not match expected containerized services." >&2
  exit 1
fi

created="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
jq -n \
  --arg version "$VERSION" \
  --arg commit "$COMMIT" \
  --arg created "$created" \
  --arg sourceCiRunId "$SOURCE_CI_RUN_ID" \
  --arg sourceCiRunUrl "$SOURCE_CI_RUN_URL" \
  --slurpfile services "$services_tmp" \
  '{
    version: $version,
    commit: $commit,
    created: $created,
    sourceCiRunId: $sourceCiRunId,
    sourceCiRunUrl: $sourceCiRunUrl,
    services: $services[0]
  }' > "$OUTPUT"

rm -f "$services_tmp"

echo "Wrote release manifest for ${service_count} service(s) to $OUTPUT."
