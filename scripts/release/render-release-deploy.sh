#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: render-release-deploy.sh --source-dir DIR --release-manifest FILE --manifest-sha256 SHA --output FILE

Renders the deployable Asterism manifest for a release by overlaying the
published image digests and release metadata onto the repo's deploy tree.
EOF
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SOURCE_DIR="${SOURCE_DIR:-deploy}"
RELEASE_MANIFEST=""
MANIFEST_SHA256=""
OUTPUT=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --source-dir)
      SOURCE_DIR="$2"
      shift 2
      ;;
    --release-manifest)
      RELEASE_MANIFEST="$2"
      shift 2
      ;;
    --manifest-sha256)
      MANIFEST_SHA256="$2"
      shift 2
      ;;
    --output)
      OUTPUT="$2"
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

if [[ -z "$RELEASE_MANIFEST" || -z "$MANIFEST_SHA256" || -z "$OUTPUT" ]]; then
  usage >&2
  exit 2
fi

if [[ ! -f "$RELEASE_MANIFEST" ]]; then
  echo "Release manifest does not exist: $RELEASE_MANIFEST" >&2
  exit 1
fi

if [[ "$SOURCE_DIR" = /* ]]; then
  SOURCE_PATH="$SOURCE_DIR"
else
  SOURCE_PATH="$REPO_ROOT/$SOURCE_DIR"
fi
if [[ ! -d "$SOURCE_PATH" ]]; then
  echo "Release source directory does not exist: $SOURCE_PATH" >&2
  exit 1
fi

version="$(jq -r '.version // empty' "$RELEASE_MANIFEST")"
commit="$(jq -r '.commit // empty' "$RELEASE_MANIFEST")"
services_json="$(jq -c '.services // []' "$RELEASE_MANIFEST")"

if [[ -z "$version" || -z "$commit" ]]; then
  echo "Release manifest is missing version or commit." >&2
  exit 1
fi

if ! jq -e 'type == "array" and length > 0 and all(.[]; (.service | type == "string" and length > 0) and (.image | type == "string" and length > 0) and (.digest | test("^sha256:[0-9a-fA-F]{64}$")) and (.version | type == "string" and length > 0))' <<< "$services_json" > /dev/null; then
  echo "Release manifest services must include service, image, digest, and version fields." >&2
  exit 1
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

ln -s "$SOURCE_PATH" "$tmp_dir/deploy"

images_block="$(
  jq -r '
    .[] |
    "  - name: \(.image)\n" +
    "    newName: \(.image)\n" +
    "    newTag: \(.version)\n" +
    "    digest: \(.digest)"
  ' <<< "$services_json"
)"

mkdir -p "$(dirname "$OUTPUT")"
cat > "$tmp_dir/kustomization.yaml" <<EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: app-asterism

resources:
  - deploy

labels:
  - pairs:
      app: asterism

images:
$images_block

patches:
  - target:
      group: apps
      version: v1
      kind: Deployment
      labelSelector: app.kubernetes.io/part-of=asterism
    patch: |-
      - op: add
        path: /spec/template/metadata/annotations/asterism.dev~1release-version
        value: "$version"
      - op: add
        path: /spec/template/metadata/annotations/asterism.dev~1release-commit
        value: "$commit"
      - op: add
        path: /spec/template/metadata/annotations/asterism.dev~1release-manifest-sha256
        value: "$MANIFEST_SHA256"
EOF

kustomize build --load-restrictor=LoadRestrictionsNone "$tmp_dir" > "$OUTPUT"

echo "Wrote release deployment manifest to $OUTPUT."
