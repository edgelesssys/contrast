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

# Uninstall existing GPU operator, if exists
helm_release=$(helm list -n gpu-operator 2>/dev/null | (grep gpu-operator || true) | awk '{print $1}')
if [[ -n $helm_release ]]; then
  # Skip installation if the installed version matches the desired version.
  current_version=$(helm list -n gpu-operator 2>/dev/null | grep gpu-operator | awk '{print $10}')
  if [[ $current_version == "$GPU_OPERATOR_VERSION" ]]; then
    echo "GPU Operator version $GPU_OPERATOR_VERSION is already installed. Skipping installation."
    exit 0
  fi
  helm delete -n gpu-operator "$helm_release"
fi
kubectl delete crd nvidiadrivers.nvidia.com --ignore-not-found

# Add the NVIDIA Helm repository
helm repo add nvidia https://helm.ngc.nvidia.com/nvidia && helm repo update

# Install the GPU Operator
# The Kata sandbox plugin defaults to the pgpu alias when P_GPU_ALIAS is unset.
# An explicitly empty value disables the alias and exposes model-specific resources.
helm install --wait --generate-name \
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

until [[ "$(
  kubectl get nodes -l nvidia.com/gpu.present -o json |
    jq '[.items[] | .status.allocatable
        | to_entries[] | select(.key | startswith("nvidia.com/"))
        | select(.value != "0")] | length'
)" -gt 0 ]]; do
  echo 'Waiting for GPU to become available...'
  sleep 5
done
