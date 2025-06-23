#!/usr/bin/env bash
# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

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

# Extract used runtime classes from running pods
touch usedRuntimeClasses
for resource in "${resourcesToCheck[@]}"; do
  kubectl get "${resource}" --all-namespaces -o jsonpath='{.items[*].spec.runtimeClassName} {.items[*].spec.template.spec.runtimeClassName} {.items[*].spec.jobTemplate.spec.template.spec.runtimeClassName}' 2>/dev/null |
    tr ' ' '\n' >>usedRuntimeClasses
done

# Extract runtime class names from running node installers
kubectl get pods --all-namespaces -o jsonpath='{.items[?(@.metadata.annotations.contrast\.edgeless\.systems/pod-role=="contrast-node-installer")].metadata.name}' |
  tr ' ' '\n' |
  grep -o "contrast-cc-.\+" |
  sed "s/-nodeinstaller.*//g" >>usedRuntimeClasses || true
sort -u usedRuntimeClasses -o usedRuntimeClasses

# Unused runtime classes is the difference between all runtime classes and the used ones
mapfile -t unusedRuntimeClasses < <(
  comm -13 usedRuntimeClasses <(
    {
      # Get all existing runtime classes that start with "contrast-cc"
      kubectl get runtimeclass -o jsonpath='{.items[*].metadata.name}' |
        tr ' ' '\n' |
        grep '^contrast-cc' || true
      # Get all (maybe already deleted) runtime classes with a reference in /opt/edgeless
      ls -1 "${HOST_MOUNT:?}/opt/edgeless"
    } | sort -u
  )
)

for runtimeClass in "${unusedRuntimeClasses[@]}"; do
  # Delete unused runtime classes
  echo "Deleting runtimeclass ${runtimeClass} ..."
  kubectl delete runtimeclass "${runtimeClass}" || true

  # Delete unused files
  if [[ -d "${HOST_MOUNT}/opt/edgeless/${runtimeClass}" ]]; then
    echo "Deleting files from ${HOST_MOUNT}/opt/edgeless/${runtimeClass} ..."
    rm -rf "${HOST_MOUNT:?}/opt/edgeless/${runtimeClass}"
  fi
  if [[ -d "${HOST_MOUNT}/var/lib/${SNAPSHOTTER}-snapshotter" ]]; then
    echo "Deleting files from ${HOST_MOUNT}/var/lib/${SNAPSHOTTER}-snapshotter/${runtimeClass} ..."
    rm -rf "${HOST_MOUNT:?}/var/lib/${SNAPSHOTTER}-snapshotter/${runtimeClass}"
  fi

  # Remove references from containerd config
  echo "Removing ${runtimeClass} from ${HOST_MOUNT}/${CONFIG} ..."
  # First try config v3 path. If this fails, try config v2 path.
  dasel delete --file "${HOST_MOUNT:?}/${CONFIG}" --indent 0 --read toml --write toml "plugins.io\.containerd\.cri\.v1\.runtime.containerd.runtimes.${runtimeClass}" ||
    dasel delete --file "${HOST_MOUNT:?}/${CONFIG}" --indent 0 --read toml --write toml "plugins.io\.containerd\.grpc\.v1\.cri.containerd.runtimes.${runtimeClass}" ||
    true

  dasel delete --file "${HOST_MOUNT:?}/${CONFIG}" --indent 0 --read toml --write toml "proxy_plugins.${SNAPSHOTTER}-${runtimeClass}" 2>/dev/null || true
done

echo "Cleanup finished"
