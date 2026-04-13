#!/usr/bin/env bash

set -euo pipefail

SERVICES='[]'

uses_nodejs() {
  [ -f "${1}/package.json" ]
}

uses_golang() {
  [ -f "${1}/go.mod" ]
}

has_dockerfile() {
  [ -f "${1}/Dockerfile" ]
}

has_kustomize_deployment() {
  [ -f "${1}/deploy/kustomization.yaml" ]
}

for service in *; do
  if [ ! -d "${service}" ]; then
    continue
  fi

  SERVICE=$(jq -nc \
    --arg id "${service}" \
    --arg path "services/${service}" \
    '{"id":$id,"path":$path,"tools":[],"containerized":false,"has_kustomize_deployment":false}')

  if uses_nodejs "${service}"; then
    SERVICE=$(jq -c '.tools += ["nodejs"]' <<< "${SERVICE}")
  fi

  if uses_golang "${service}"; then
    SERVICE=$(jq -c '.tools += ["go"]' <<< "${SERVICE}")
  fi

  if has_dockerfile "${service}"; then
    SERVICE=$(jq -c '.containerized = true' <<< "${SERVICE}")
  fi

  if has_kustomize_deployment "${service}"; then
    SERVICE=$(jq -c '.has_kustomize_deployment = true' <<< "${SERVICE}")
  fi

  SERVICES=$(jq -c --argjson service "${SERVICE}" '. += [$service]' <<< "${SERVICES}")
done

echo "${SERVICES}"
