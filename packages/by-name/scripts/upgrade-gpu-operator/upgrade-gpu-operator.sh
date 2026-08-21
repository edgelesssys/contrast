#!/usr/bin/env bash
# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

GPU_OPERATOR_VERSION=""

while [[ $# -gt 0 ]]; do
  case $1 in
  --version)
    if [[ -n ${2:-} ]]; then
      GPU_OPERATOR_VERSION="$2"
      shift 2
    else
      echo "Error: --version requires an argument"
      exit 1
    fi
    ;;
  *)
    echo "Unknown option: $1"
    exit 1
    ;;
  esac
done

if [[ -z $GPU_OPERATOR_VERSION ]]; then
  echo "Usage: $0 --version <gpu-operator-version>"
  exit 1
fi

# Reuse the existing release name if the operator is already installed. Clusters
# set up before this script used --generate-name, so their release is called
# something like gpu-operator-1787258651. Upgrading that release in place keeps
# them on one release instead of stranding the old one next to a new one.
RELEASE=gpu-operator
existing=$(helm list -n gpu-operator -o json 2>/dev/null | jq -r '.[].name' | grep '^gpu-operator' | head -1 || true)
if [[ -n $existing ]]; then
  RELEASE="$existing"
fi

# Add the NVIDIA Helm repository
helm repo add nvidia https://helm.ngc.nvidia.com/nvidia && helm repo update

# Install the GPU Operator
# The Kata sandbox plugin defaults to the pgpu alias when P_GPU_ALIAS is unset.
# An explicitly empty value disables the alias and exposes model-specific resources.
helm upgrade --install "$RELEASE" --wait \
  -n gpu-operator --create-namespace \
  nvidia/gpu-operator \
  --version="$GPU_OPERATOR_VERSION" \
  --set sandboxWorkloads.enabled=true \
  --set sandboxWorkloads.defaultWorkload=vm-passthrough \
  --set sandboxWorkloads.mode=kata \
  --set 'kataSandboxDevicePlugin.env[0].name=P_GPU_ALIAS' \
  --set 'kataSandboxDevicePlugin.env[0].value=' \
  --set nfd.enabled=true \
  --set nfd.nodefeaturerules=true

# The API server drops any ClusterPolicy field the installed CRD doesn't declare,
# without erroring, and the operator then reports itself ready having deployed no
# device plugin. Check that the settings above actually reached the live object.
clusterpolicy_matches_chart() {
  kubectl get clusterpolicy -o json |
    jq -e '.items[0].spec
           | .sandboxWorkloads.mode == "kata"
             and ([.kataSandboxDevicePlugin.env[]? | select(.name == "P_GPU_ALIAS")] | length == 1)' >/dev/null
}

if ! clusterpolicy_matches_chart; then
  # Helm's three-way merge only patches fields that differ between the previously
  # rendered manifest and the new one, so a ClusterPolicy that was pruned when it
  # was first written matches neither revision and helm leaves it untouched. The
  # CRD is current by this point, so re-applying what helm rendered converges it.
  # Helm owns .metadata and .spec here and the operator only owns .status, so this
  # doesn't contend with the operator.
  echo "WARNING: the live ClusterPolicy doesn't match the chart, reconciling it." >&2
  helm get manifest "$RELEASE" -n gpu-operator |
    yq 'select(.kind == "ClusterPolicy")' |
    kubectl apply -f -

  if ! clusterpolicy_matches_chart; then
    echo "ERROR: the ClusterPolicy still doesn't carry the kata sandbox device plugin settings." >&2
    echo "The clusterpolicies.nvidia.com CRD is probably older than the chart being installed" >&2
    echo "and is pruning them. Note that a CRD updated in place keeps its original" >&2
    echo "creationTimestamp, so check the schema rather than the age:" >&2
    kubectl get crd clusterpolicies.nvidia.com -o json |
      jq '.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties
          | {has_kataSandboxDevicePlugin: has("kataSandboxDevicePlugin"),
             has_sandboxWorkloads_mode: (.sandboxWorkloads.properties | has("mode"))}' >&2
    kubectl get clusterpolicy -o json | jq '.items[0].spec | {sandboxWorkloads, kataSandboxDevicePlugin}' >&2
    exit 1
  fi
fi

# Bounded: an operator that never deploys a device plugin would otherwise spin here
# until the CI runner's 6h job timeout, reporting only "cancelled".
deadline=$((SECONDS + 600))
until [[ "$(
  kubectl get nodes -l nvidia.com/gpu.present -o json |
    jq '[.items[] | .status.allocatable
        | to_entries[] | select(.key | startswith("nvidia.com/"))
        | select(.value != "0")] | length'
)" -gt 0 ]]; do
  if ((SECONDS > deadline)); then
    echo "ERROR: no node advertised a non-zero nvidia.com/* resource within 10m." >&2
    kubectl get ds,po -n gpu-operator >&2
    kubectl get clusterpolicy -o yaml >&2
    exit 1
  fi
  echo 'Waiting for GPU to become available...'
  sleep 5
done
