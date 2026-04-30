#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: publish-images.sh --source DIR --release-dir DIR --version VERSION --expected-services-json JSON

Loads PR-built image archives, retags them with the immutable release tag and
latest, pushes both tags to GHCR, signs the immutable digest with cosign, and
writes release image metadata files.
EOF
}

SOURCE_DIR=""
RELEASE_DIR=""
VERSION=""
EXPECTED_SERVICES_JSON="${EXPECTED_SERVICES_JSON:-}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --source)
      SOURCE_DIR="$2"
      shift 2
      ;;
    --release-dir)
      RELEASE_DIR="$2"
      shift 2
      ;;
    --version)
      VERSION="$2"
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

if [[ -z "$SOURCE_DIR" || -z "$RELEASE_DIR" || -z "$VERSION" || -z "$EXPECTED_SERVICES_JSON" ]]; then
  usage >&2
  exit 2
fi

if [[ ! -d "$SOURCE_DIR" ]]; then
  echo "Release source directory does not exist: $SOURCE_DIR" >&2
  exit 1
fi

if ! jq -e 'type == "array" and length > 0 and all(.[]; (.id | type == "string") and (.id | length > 0))' <<< "$EXPECTED_SERVICES_JSON" > /dev/null; then
  echo "Expected services JSON must be a non-empty array of service objects with id fields." >&2
  exit 1
fi

mkdir -p "$RELEASE_DIR"
mapfile -t services < <(jq -r '.[].id' <<< "$EXPECTED_SERVICES_JSON" | sort)

for service in "${services[@]}"; do
  archive="$SOURCE_DIR/image-${service}.tar"
  metadata="$SOURCE_DIR/image-metadata-${service}.json"
  sbom="$SOURCE_DIR/sbom-${service}.spdx.json"

  if [[ ! -s "$archive" ]]; then
    echo "Missing image archive for ${service}: $archive" >&2
    exit 1
  fi
  if [[ ! -s "$metadata" ]]; then
    echo "Missing image metadata for ${service}: $metadata" >&2
    exit 1
  fi
  if [[ ! -s "$sbom" ]]; then
    echo "Missing SBOM for ${service}: $sbom" >&2
    exit 1
  fi

  image="$(jq -r '.image' "$metadata")"
  source_tag="$(jq -r '.tag' "$metadata")"

  if [[ -z "$image" || "$image" == "null" || -z "$source_tag" || "$source_tag" == "null" ]]; then
    echo "Invalid image metadata for ${service}: $metadata" >&2
    exit 1
  fi

  docker load -i "$archive"
  docker image inspect "${image}:${source_tag}" > /dev/null
  docker tag "${image}:${source_tag}" "${image}:${VERSION}"
  docker tag "${image}:${source_tag}" "${image}:latest"

  push_output="$(docker push "${image}:${VERSION}")"
  printf '%s\n' "$push_output"
  digest="$(awk '/digest:/ {print $3; exit}' <<< "$push_output")"

  if [[ -z "$digest" || "$digest" != sha256:* ]]; then
    echo "Could not capture pushed digest for ${image}:${VERSION}" >&2
    exit 1
  fi

  docker push "${image}:latest"
  cosign sign --yes "${image}@${digest}"

  jq -n \
    --arg service "$service" \
    --arg image "$image" \
    --arg digest "$digest" \
    --arg version "$VERSION" \
    --arg releaseTag "$VERSION" \
    --arg latestTag "latest" \
    --arg sourceTag "$source_tag" \
    '{
      service: $service,
      image: $image,
      digest: $digest,
      version: $version,
      releaseTag: $releaseTag,
      latestTag: $latestTag,
      sourceTag: $sourceTag
    }' > "$RELEASE_DIR/image-metadata-${service}.json"
done

echo "Published, tagged, signed, and recorded ${#services[@]} image(s)."
