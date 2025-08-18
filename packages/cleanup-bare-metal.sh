#!/usr/bin/env bash
# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

echo "Starting cleanup"

declare configFile
if [[ -f "${HOST_MOUNT:?}/var/lib/rancher/k3s/agent/etc/containerd/config.toml.tmpl" ]]; then
  configFile="${HOST_MOUNT:?}/var/lib/rancher/k3s/agent/etc/containerd/config.toml.tmpl"
elif [[ -f "${HOST_MOUNT:?}/etc/containerd/config.toml" ]]; then
  configFile="${HOST_MOUNT:?}/etc/containerd/config.toml"
else
  echo "No containerd config file found. Exiting."
  exit 1
fi
echo "Using containerd config file: ${configFile}"

configVersion=$(dasel --file "${configFile}" --read toml version)
if [[ ${configVersion} != "2" && ${configVersion} != "3" ]]; then
  echo "Unsupported containerd config version: ${configVersion}. Exiting."
  exit 1
fi
echo "Containerd config version: ${configVersion}"

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
  echo "Removing ${runtimeClass} from ${configFile} ..."
  case "${configVersion}" in
  "2")
    dasel delete --file "${configFile}" --indent 0 --read toml --write toml "plugins.io\.containerd\.grpc\.v1\.cri.containerd.runtimes.${runtimeClass}" || true
    ;;
  "3")
    dasel delete --file "${configFile}" --indent 0 --read toml --write toml "plugins.io\.containerd\.cri\.v1\.runtime.containerd.runtimes.${runtimeClass}" || true
    ;;
  esac

  dasel delete --file "${configFile}" --indent 0 --read toml --write toml "proxy_plugins.${SNAPSHOTTER}-${runtimeClass}" 2>/dev/null || true
done

echo "Cleanup finished"
