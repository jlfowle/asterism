#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: normalize-artifacts.sh --source DIR --dest DIR --expected-services-json JSON

Copies downloaded CI build artifacts into a flat release-source directory and
fails if any expected containerized service is missing its image archive, SBOM,
or image metadata file.
EOF
}

SOURCE_DIR=""
DEST_DIR=""
EXPECTED_SERVICES_JSON="${EXPECTED_SERVICES_JSON:-}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --source)
      SOURCE_DIR="$2"
      shift 2
      ;;
    --dest)
      DEST_DIR="$2"
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

if [[ -z "$SOURCE_DIR" || -z "$DEST_DIR" || -z "$EXPECTED_SERVICES_JSON" ]]; then
  usage >&2
  exit 2
fi

if [[ ! -d "$SOURCE_DIR" ]]; then
  echo "Artifact source directory does not exist: $SOURCE_DIR" >&2
  exit 1
fi

if ! jq -e 'type == "array" and length > 0 and all(.[]; (.id | type == "string") and (.id | length > 0))' <<< "$EXPECTED_SERVICES_JSON" > /dev/null; then
  echo "Expected services JSON must be a non-empty array of service objects with id fields." >&2
  exit 1
fi

find_unique_file() {
  local name="$1"
  local matches=()
  mapfile -t matches < <(find "$SOURCE_DIR" -type f -name "$name" | sort)

  if [[ ${#matches[@]} -eq 0 ]]; then
    echo "Missing required artifact: $name" >&2
    return 1
  fi

  if [[ ${#matches[@]} -gt 1 ]]; then
    echo "Found multiple artifacts named $name:" >&2
    printf '  %s\n' "${matches[@]}" >&2
    return 1
  fi

  printf '%s\n' "${matches[0]}"
}

rm -rf "$DEST_DIR"
mkdir -p "$DEST_DIR"

mapfile -t services < <(jq -r '.[].id' <<< "$EXPECTED_SERVICES_JSON" | sort)

for service in "${services[@]}"; do
  archive_name="image-${service}.tar"
  metadata_name="image-metadata-${service}.json"
  sbom_name="sbom-${service}.spdx.json"

  archive_path="$(find_unique_file "$archive_name")"
  metadata_path="$(find_unique_file "$metadata_name")"
  sbom_path="$(find_unique_file "$sbom_name")"

  cp "$archive_path" "$DEST_DIR/$archive_name"
  cp "$metadata_path" "$DEST_DIR/$metadata_name"
  cp "$sbom_path" "$DEST_DIR/$sbom_name"

  test -s "$DEST_DIR/$archive_name"
  test -s "$DEST_DIR/$metadata_name"
  test -s "$DEST_DIR/$sbom_name"

  jq -e \
    --arg service "$service" \
    '.service == $service and (.image | type == "string" and length > 0) and (.tag | type == "string" and length > 0) and (.version | type == "string" and length > 0)' \
    "$DEST_DIR/$metadata_name" > /dev/null
done

jq -n --argjson services "$EXPECTED_SERVICES_JSON" '{containerized: $services}' > "$DEST_DIR/expected-services.json"

echo "Normalized release source artifacts for ${#services[@]} service(s) into $DEST_DIR."
