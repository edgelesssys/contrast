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
# Deliberately not guarded with `|| true` or `2>/dev/null`: if helm itself fails,
# stop. Swallowing the error would leave $existing empty, fall through to the
# default name, and install a second release beside the one already there.
existing=$(helm list -n gpu-operator -o json | jq -r '.[].name | select(startswith("gpu-operator"))')

RELEASE=gpu-operator
if [[ -n $existing ]]; then
  if [[ $(wc -l <<<"$existing") -gt 1 ]]; then
    # Two releases means someone installed a second one by hand. Picking either
    # silently would upgrade one and leave the other shadowing it on the same node.
    echo "ERROR: more than one gpu-operator helm release in the gpu-operator namespace:" >&2
    echo "$existing" >&2
    echo "Remove the stale one before running this." >&2
    exit 1
  fi
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
# device plugin. Assert the settings above actually survived, so a stale CRD fails
# here instead of silently producing a GPU-less node.
if ! kubectl get clusterpolicy -o json |
  jq -e '.items[0].spec
         | .sandboxWorkloads.mode == "kata"
           and ([.kataSandboxDevicePlugin.env[]? | select(.name == "P_GPU_ALIAS")] | length == 1)' >/dev/null; then
  echo "ERROR: the ClusterPolicy did not retain the kata sandbox device plugin settings." >&2
  echo "This usually means clusterpolicies.nvidia.com is older than the chart being installed." >&2
  echo -n "  CRD created: " >&2
  kubectl get crd clusterpolicies.nvidia.com -o jsonpath='{.metadata.creationTimestamp}{"\n"}' >&2
  kubectl get clusterpolicy -o json | jq '.items[0].spec | {sandboxWorkloads, kataSandboxDevicePlugin}' >&2
  exit 1
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
