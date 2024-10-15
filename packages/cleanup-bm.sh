#!/usr/bin/env bash
# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

set -euo pipefail

echo "Starting cleanup"

mapfile -t runtimeClasses < <(kubectl get runtimeclass -o jsonpath='{.items[*].metadata.name}' | tr ' ' '\n' | grep '^contrast-cc')

declare -a usedRuntimeClasses=()
declare -a unusedRuntimeClasses=()

resourcesToCheck=(
  "pods"
  "deployments"
  "statefulsets"
  "daemonsets"
  "replicasets"
  "jobs"
  "cronjobs"
)

for resource in "${resourcesToCheck[@]}"; do
  rc=$(
    kubectl get "${resource}" --all-namespaces -o jsonpath='{.items[*].spec.runtimeClassName}' 2>/dev/null |
      tr ' ' '\n'
  )
  if [ -n "${rc}" ]; then
    usedRuntimeClasses+=("${rc}")
  fi
done

if [ "${#usedRuntimeClasses[@]}" -eq 0 ]; then
  unusedRuntimeClasses=("${runtimeClasses[@]}")
else
  mapfile -t unusedRuntimeClasses < <(
    printf "%s\n" "${runtimeClasses[@]}" |
      grep -v -F -x -f <(printf "%s\n" "${usedRuntimeClasses[@]}")
  )
fi

for runtimeClass in "${unusedRuntimeClasses[@]}"; do
  # Delete unused runtime classes
  echo "Deleting runtimeclass ${runtimeClass} ..."
  kubectl delete runtimeclass "${runtimeClass}"

  # Delete unused binaries
  path="${OPTEDGELESS}/${runtimeClass}"
  if [ -d "${path}" ]; then
    echo "Deleting binaries from ${OPTEDGELESS}/${runtimeClass} ..."
    rm -rf "${path}"
  fi

  # Remove references from containerd config
  echo "Removing ${runtimeClass} from ${CONFIG} ..."
  dasel delete --file "${CONFIG}" --indent 0 --read toml --write toml "plugins.io\.containerd\.grpc\.v1\.cri.containerd.runtimes.${runtimeClass}" 2>/dev/null
  dasel delete --file "${CONFIG}" --indent 0 --read toml --write toml "proxy_plugins.${SNAPSHOTTER}-${runtimeClass}" 2>/dev/null
done
