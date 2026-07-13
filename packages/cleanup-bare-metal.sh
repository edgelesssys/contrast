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

configVersion=$(yq -p toml -o yaml '.version' "${configFile}")
if [[ ${configVersion} != "2" && ${configVersion} != "3" ]]; then
  echo "Unsupported containerd config version: ${configVersion}. Exiting."
  exit 1
fi
echo "Containerd config version: ${configVersion}"

echo "Deleting old containernd config backups ..."
find "$(dirname "${configFile}")" -maxdepth 1 -type f -name "$(basename "${configFile}").*.bak" -mtime +5 -delete

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
{
  # TODO(burgerdev): checking the annotation is a fallback for older runtimes, can be removed in Q4 2026.
  kubectl get pods --all-namespaces -o jsonpath='{.items[?(@.metadata.annotations.contrast\.edgeless\.systems/pod-role=="contrast-node-installer")].metadata.name}'
  kubectl get pods --all-namespaces --selector contrast.edgeless.systems/pod-role=contrast-node-installer -o jsonpath='{.items[*].metadata.name}'
} |
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
    yq -i -p toml -o toml "del(.plugins[\"io.containerd.grpc.v1.cri\"].containerd.runtimes[\"${runtimeClass}\"])" "${configFile}" || true
    ;;
  "3")
    yq -i -p toml -o toml "del(.plugins[\"io.containerd.cri.v1.runtime\"].containerd.runtimes[\"${runtimeClass}\"])" "${configFile}" || true
    ;;
  esac

  yq -i -p toml -o toml "del(.proxy_plugins[\"${SNAPSHOTTER}-${runtimeClass}\"])" "${configFile}" 2>/dev/null || true
done

# Clean up potential leftovers in the default namespace.
kubectl delete configmaps --selector app.kubernetes.io/managed-by=contrast.edgeless.systems || true

echo "Cleanup finished"

if nsenter -t 1 -m sh -c "command -v k3s >/dev/null 2>&1"; then
  installed_version=$(nsenter -t 1 -m k3s --version | head -n 1 | cut -d ' ' -f3)
  echo "Installed k3s version: ${installed_version}"
  if [[ $installed_version != "$K3S_VERSION" ]]; then
    echo "Updating k3s to version ${K3S_VERSION} ..."
    mkdir -p /host/etc/rancher/k3s
    # NOTE:
    # If you change something here, make sure to change dev-docs/e2e/bare-metal-runner.md, too!
    cat >/host/etc/rancher/k3s/config.yaml <<EOF
write-kubeconfig-mode: "0640"
write-kubeconfig-group: sudo
disable:
  - local-storage
kubelet-arg:
  - "runtime-request-timeout=5m"
node-label:
  - ci.contrast.edgeless.systems/main-runner=true
embedded-registry: true
EOF
    nsenter -t 1 -m -n sh -c "curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION=${K3S_VERSION} sh -"
  else
    echo "k3s is already at version ${K3S_VERSION}. No update needed."
  fi
else
  echo "k3s command not found. Skipping k3s version check."
fi
