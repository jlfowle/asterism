#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: resolve-release-state.sh --repo OWNER/REPO --commit SHA --fallback-version VERSION --manifest-path FILE [--token TOKEN] [--releases-json FILE]

Looks for an existing published GitHub release whose release-manifest.json
matches the supplied commit. If found, writes that manifest to --manifest-path
and reports the release version. Otherwise, returns the fallback version.
EOF
}

REPO=""
COMMIT=""
FALLBACK_VERSION=""
MANIFEST_PATH=""
TOKEN="${TOKEN:-}"
RELEASES_JSON=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --repo)
      REPO="$2"
      shift 2
      ;;
    --commit)
      COMMIT="$2"
      shift 2
      ;;
    --fallback-version)
      FALLBACK_VERSION="$2"
      shift 2
      ;;
    --manifest-path)
      MANIFEST_PATH="$2"
      shift 2
      ;;
    --token)
      TOKEN="$2"
      shift 2
      ;;
    --releases-json)
      RELEASES_JSON="$2"
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

if [[ -z "$REPO" || -z "$COMMIT" || -z "$FALLBACK_VERSION" || -z "$MANIFEST_PATH" ]]; then
  usage >&2
  exit 2
fi

if [[ -z "$RELEASES_JSON" && -z "$TOKEN" ]]; then
  echo "--token is required unless --releases-json is provided." >&2
  exit 2
fi

mkdir -p "$(dirname "$MANIFEST_PATH")"

download_asset() {
  local url="$1"
  local dest="$2"

  case "$url" in
    file://*)
      local path="${url#file://}"
      if [[ -z "$path" ]]; then
        echo "Release asset URL is missing a local path: $url" >&2
        return 1
      fi
      cp "$path" "$dest"
      ;;
    /*)
      cp "$url" "$dest"
      ;;
    *)
      local headers=(-H "Accept: application/octet-stream" -H "X-GitHub-Api-Version: 2022-11-28")
      if [[ -n "$TOKEN" ]]; then
        headers+=(-H "Authorization: Bearer $TOKEN")
      fi
      curl -fsSL "${headers[@]}" "$url" -o "$dest"
      ;;
  esac
}

found=0
resolved_version=""
resolved_release_tag=""
resolved_release_url=""

process_release_json() {
  local release_json="$1"

  if [[ "$(jq -r '.draft // false' <<< "$release_json")" == "true" ]]; then
    return 0
  fi

  local asset_url
  asset_url="$(jq -r '.assets[]? | select(.name == "release-manifest.json") | .browser_download_url' <<< "$release_json" | head -n 1)"
  if [[ -z "$asset_url" || "$asset_url" == "null" ]]; then
    return 0
  fi

  local tmp_manifest
  tmp_manifest="$(mktemp)"
  if ! download_asset "$asset_url" "$tmp_manifest"; then
    rm -f "$tmp_manifest"
    return 1
  fi

  local manifest_commit
  manifest_commit="$(jq -r '.commit // empty' "$tmp_manifest")"
  if [[ "$manifest_commit" == "$COMMIT" ]]; then
    resolved_version="$(jq -r '.version // empty' "$tmp_manifest")"
    if [[ -z "$resolved_version" ]]; then
      echo "Matched release manifest is missing a version." >&2
      rm -f "$tmp_manifest"
      exit 1
    fi

    resolved_release_tag="$(jq -r '.tag_name // empty' <<< "$release_json")"
    resolved_release_url="$(jq -r '.html_url // empty' <<< "$release_json")"
    mv "$tmp_manifest" "$MANIFEST_PATH"
    found=1
    return 0
  fi

  rm -f "$tmp_manifest"
}

process_release_page() {
  local page_json="$1"
  local release_json

  while IFS= read -r release_json; do
    process_release_json "$release_json"
    if [[ "$found" -eq 1 ]]; then
      return 0
    fi
  done < <(jq -c '.[]' <<< "$page_json")
}

if [[ -n "$RELEASES_JSON" ]]; then
  process_release_page "$(cat "$RELEASES_JSON")"
else
  page=1
  while true; do
    page_json="$(
      curl -fsSL \
        -H "Accept: application/vnd.github+json" \
        -H "Authorization: Bearer $TOKEN" \
        -H "X-GitHub-Api-Version: 2022-11-28" \
        "https://api.github.com/repos/$REPO/releases?per_page=100&page=$page"
    )"

    process_release_page "$page_json"
    if [[ "$found" -eq 1 ]]; then
      break
    fi

    if [[ "$(jq 'length' <<< "$page_json")" -lt 100 ]]; then
      break
    fi

    page=$((page + 1))
  done
fi

if [[ "$found" -eq 1 ]]; then
  result="$(
    jq -n \
      --arg version "$resolved_version" \
      --arg release_tag "$resolved_release_tag" \
      --arg release_url "$resolved_release_url" \
      --argjson release_exists true \
      '{
        release_exists: $release_exists,
        version: $version,
        release_tag: $release_tag,
        release_url: $release_url
      }'
  )"
  printf '%s\n' "$result"

  if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
    {
      echo "release_exists=true"
      echo "version=$resolved_version"
      if [[ -n "$resolved_release_tag" ]]; then
        echo "release_tag=$resolved_release_tag"
      fi
      if [[ -n "$resolved_release_url" ]]; then
        echo "release_url=$resolved_release_url"
      fi
    } >> "$GITHUB_OUTPUT"
  fi

  echo "Reusing existing release $resolved_version for commit $COMMIT." >&2
else
  rm -f "$MANIFEST_PATH"
  result="$(
    jq -n \
      --arg version "$FALLBACK_VERSION" \
      --argjson release_exists false \
      '{
        release_exists: $release_exists,
        version: $version,
        release_tag: "",
        release_url: ""
      }'
  )"
  printf '%s\n' "$result"

  if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
    {
      echo "release_exists=false"
      echo "version=$FALLBACK_VERSION"
    } >> "$GITHUB_OUTPUT"
  fi

  echo "No existing release found for commit $COMMIT; using $FALLBACK_VERSION." >&2
fi
