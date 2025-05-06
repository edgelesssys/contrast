#!/usr/bin/env bash
# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

set -euo pipefail

echo "Starting cleanup"

resourcesToCheck=(
  "pods"
  "deployments"
  "statefulsets"
  "daemonsets"
  "replicasets"
  "jobs"
  "cronjobs"
)

touch usedRuntimeClasses
for resource in "${resourcesToCheck[@]}"; do
  kubectl get "${resource}" --all-namespaces -o jsonpath='{.items[*].spec.runtimeClassName} {.items[*].spec.template.spec.runtimeClassName} {.items[*].spec.jobTemplate.spec.template.spec.runtimeClassName}' 2>/dev/null |
    tr ' ' '\n' >>usedRuntimeClasses
done

kubectl get pods --all-namespaces -o jsonpath='{.items[?(@.metadata.annotations.contrast\.edgeless\.systems/pod-role=="contrast-node-installer")].spec.containers[0].args[1]}' |
  tr ' ' '\n' |
  grep -o "contrast-cc-.\+" >>usedRuntimeClasses || true
sort -u usedRuntimeClasses -o usedRuntimeClasses

mapfile -t unusedRuntimeClasses < <(
  comm -13 usedRuntimeClasses <(
    kubectl get runtimeclass -o jsonpath='{.items[*].metadata.name}' |
      tr ' ' '\n' |
      grep '^contrast-cc' |
      sort -u
  )
)

for runtimeClass in "${unusedRuntimeClasses[@]}"; do
  # Delete unused runtime classes
  echo "Deleting runtimeclass ${runtimeClass} ..."
  kubectl delete runtimeclass "${runtimeClass}"

  # Delete unused files
  if [ -d "${OPTEDGELESS}/${runtimeClass}" ]; then
    echo "Deleting files from ${OPTEDGELESS}/${runtimeClass} ..."
    rm -rf "${OPTEDGELESS:?}/${runtimeClass}"
  fi
  if [ -d "/var/lib/${SNAPSHOTTER}-snapshotter" ]; then
    echo "Deleting files from /var/lib/${SNAPSHOTTER}-snapshotter/${runtimeClass} ..."
    rm -rf "/var/lib/${SNAPSHOTTER}-snapshotter/${runtimeClass}"
  fi

  # Remove references from containerd config
  echo "Removing ${runtimeClass} from ${CONFIG} ..."
  # First try config v3 path. If this fails, try config v2 path.
  dasel delete --file "${CONFIG}" --indent 0 --read toml --write toml "plugins.io\.containerd\.cri\.v1\.runtime.containerd.runtimes.${runtimeClass}" ||
    dasel delete --file "${CONFIG}" --indent 0 --read toml --write toml "plugins.io\.containerd\.grpc\.v1\.cri.containerd.runtimes.${runtimeClass}" ||
    true

  dasel delete --file "${CONFIG}" --indent 0 --read toml --write toml "proxy_plugins.${SNAPSHOTTER}-${runtimeClass}" 2>/dev/null || true
done

# Fix the state for removed snapshotters.
cleanup-images

echo "Cleanup finished"
