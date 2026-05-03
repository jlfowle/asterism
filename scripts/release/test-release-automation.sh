#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

expected='[{"id":"pfsense","containerized":true}]'

source_dir="$TMP_DIR/source"
mkdir -p "$source_dir/nested/workspace" "$source_dir/runner/_temp"
printf 'image archive\n' > "$source_dir/runner/_temp/image-pfsense.tar"
printf 'sbom\n' > "$source_dir/nested/workspace/sbom-pfsense.spdx.json"
jq -n \
  --arg service pfsense \
  --arg image ghcr.io/jlfowle/asterism-pfsense \
  --arg tag v1.2.3-pr.1-commit.abcdef0 \
  '{service: $service, image: $image, tag: $tag, version: $tag}' \
  > "$source_dir/nested/workspace/image-metadata-pfsense.json"

normalized_dir="$TMP_DIR/normalized"
"$SCRIPT_DIR/normalize-artifacts.sh" \
  --source "$source_dir" \
  --dest "$normalized_dir" \
  --expected-services-json "$expected"

test -s "$normalized_dir/image-pfsense.tar"
test -s "$normalized_dir/image-metadata-pfsense.json"
test -s "$normalized_dir/sbom-pfsense.spdx.json"

missing_dir="$TMP_DIR/missing"
mkdir -p "$missing_dir"
cp "$source_dir/runner/_temp/image-pfsense.tar" "$missing_dir/"
cp "$source_dir/nested/workspace/image-metadata-pfsense.json" "$missing_dir/"
if "$SCRIPT_DIR/normalize-artifacts.sh" --source "$missing_dir" --dest "$TMP_DIR/should-not-exist" --expected-services-json "$expected" >/dev/null 2>&1; then
  echo "normalize-artifacts.sh unexpectedly succeeded without an SBOM." >&2
  exit 1
fi

release_dir="$TMP_DIR/release"
mkdir -p "$release_dir"
jq -n \
  --arg service pfsense \
  --arg image ghcr.io/jlfowle/asterism-pfsense \
  --arg digest sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef \
  --arg version v1.2.3 \
  '{service: $service, image: $image, digest: $digest, version: $version}' \
  > "$release_dir/image-metadata-pfsense.json"

manifest="$TMP_DIR/release-manifest.json"
"$SCRIPT_DIR/build-release-manifest.sh" \
  --release-dir "$release_dir" \
  --output "$manifest" \
  --version v1.2.3 \
  --commit abcdef0123456789 \
  --expected-services-json "$expected"

jq -e '.services | length == 1' "$manifest" > /dev/null
jq -e '.services[0].digest == "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"' "$manifest" > /dev/null

if "$SCRIPT_DIR/build-release-manifest.sh" --release-dir "$TMP_DIR" --output "$TMP_DIR/empty.json" --version v1.2.3 --commit abc --expected-services-json "$expected" >/dev/null 2>&1; then
  echo "build-release-manifest.sh unexpectedly succeeded without image metadata." >&2
  exit 1
fi

resolve_dir="$TMP_DIR/resolve"
mkdir -p "$resolve_dir"
cat > "$resolve_dir/release-manifest.json" <<'EOF'
{"version":"v1.2.3","commit":"abcdef0123456789","created":"2026-05-03T00:00:00Z","sourceCiRunId":"42","sourceCiRunUrl":"https://example.invalid/run/42","services":[]}
EOF
jq -n \
  --arg manifest_url "file://$resolve_dir/release-manifest.json" \
  '[{
      tag_name: "v1.2.3",
      html_url: "https://example.invalid/releases/v1.2.3",
      draft: false,
      prerelease: false,
      assets: [
        {
          name: "release-manifest.json",
          browser_download_url: $manifest_url
        }
      ]
    }]' \
  > "$resolve_dir/releases.json"

resolved_manifest="$TMP_DIR/resolved-release-manifest.json"
resolved_output="$("$SCRIPT_DIR/resolve-release-state.sh" \
  --repo jlfowle/asterism \
  --commit abcdef0123456789 \
  --fallback-version v9.9.9 \
  --manifest-path "$resolved_manifest" \
  --releases-json "$resolve_dir/releases.json")"

echo "$resolved_output" | jq -e '.release_exists == true and .version == "v1.2.3"' > /dev/null
cmp -s "$resolved_manifest" "$resolve_dir/release-manifest.json"

fallback_manifest="$TMP_DIR/fallback-release-manifest.json"
fallback_output="$("$SCRIPT_DIR/resolve-release-state.sh" \
  --repo jlfowle/asterism \
  --commit feedfacefeedface \
  --fallback-version v9.9.9 \
  --manifest-path "$fallback_manifest" \
  --releases-json "$resolve_dir/releases.json")"

echo "$fallback_output" | jq -e '.release_exists == false and .version == "v9.9.9"' > /dev/null
if [[ -e "$fallback_manifest" ]]; then
  echo "resolve-release-state.sh unexpectedly wrote a manifest for a missing release." >&2
  exit 1
fi

cd_repo="$TMP_DIR/os-config"
mkdir -p "$cd_repo/app/asterism"
cat > "$cd_repo/app/asterism/kustomization.yaml" <<'EOF'
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: app-asterism
resources:
  - github.com/jlfowle/asterism//deploy?ref=v0.1.0
labels:
  - pairs:
      app: asterism
EOF

"$SCRIPT_DIR/promote-os-config.py" \
  --repo-dir "$cd_repo" \
  --version v1.2.3 \
  --source-repo jlfowle/asterism \
  --commit abcdef0123456789 \
  --manifest-sha256 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef

grep -q 'github.com/jlfowle/asterism//deploy?ref=v1.2.3' "$cd_repo/app/asterism/kustomization.yaml"
grep -q 'asterism.dev~1release-version' "$cd_repo/app/asterism/kustomization.yaml"
grep -q '0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef' "$cd_repo/app/asterism/kustomization.yaml"

before_sha="$(sha256sum "$cd_repo/app/asterism/kustomization.yaml" | awk '{print $1}')"
"$SCRIPT_DIR/promote-os-config.py" \
  --repo-dir "$cd_repo" \
  --version v1.2.3 \
  --source-repo jlfowle/asterism \
  --commit abcdef0123456789 \
  --manifest-sha256 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
after_sha="$(sha256sum "$cd_repo/app/asterism/kustomization.yaml" | awk '{print $1}')"

if [[ "$before_sha" != "$after_sha" ]]; then
  echo "promote-os-config.py changed an already up-to-date overlay." >&2
  exit 1
fi

echo "Release automation checks passed."
