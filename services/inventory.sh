#!/usr/bin/env bash

SERVICES='[]'

uses_nodejs() {
  if [ -f "${1}/package.json" ]; then
    return 0
  else
    return 1
  fi
}

for service in *; do
  if [ ! -d "${service}" ]; then
    continue
  fi

  SERVICE='{"id":"'${service}'","tools":[]}'
  
  if uses_nodejs $service; then
    SERVICE=$(jq -c '.tools += ["nodejs"]' <<< $SERVICE)
  fi

  SERVICES=$(jq -c '. += ['${SERVICE}']' <<< $SERVICES)
done

echo $SERVICES