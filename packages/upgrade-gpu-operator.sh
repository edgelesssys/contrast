#!/usr/bin/env bash
# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

BLACKWELL=false
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
  --blackwell)
    BLACKWELL=true
    shift
    ;;
  *)
    echo "Unknown option: $1"
    exit 1
    ;;
  esac
done

if [[ -z $GPU_OPERATOR_VERSION ]]; then
  echo "Usage: $0 --version <gpu-operator-version> [--blackwell]"
  exit 1
fi

# Uninstall existing GPU operator, if exists
helm_release=$(helm list -n gpu-operator 2>/dev/null | (grep gpu-operator || true) | awk '{print $1}')
if [[ -n $helm_release ]]; then
  # Exit if the installed version matches the desired version
  current_version=$(helm list -n gpu-operator 2>/dev/null | grep gpu-operator | awk '{print $10}')
  if [[ $current_version == "$GPU_OPERATOR_VERSION" ]]; then
    echo "GPU Operator version $GPU_OPERATOR_VERSION is already installed. Skipping installation."
    exit 0
  fi
  helm delete -n gpu-operator "$helm_release"
fi
kubectl delete crd nvidiadrivers.nvidia.com --ignore-not-found

# Install GPU Operator
helm repo add nvidia https://helm.ngc.nvidia.com/nvidia && helm repo update

# Upstream instructions from https://github.com/kata-containers/kata-containers/pull/12257
helm install --wait --generate-name \
  -n gpu-operator --create-namespace \
  nvidia/gpu-operator \
  --version="$GPU_OPERATOR_VERSION" \
  --set sandboxWorkloads.enabled=true \
  --set sandboxWorkloads.defaultWorkload=vm-passthrough \
  --set kataManager.enabled=true \
  --set kataManager.config.runtimeClasses=null \
  --set kataManager.repository=nvcr.io/nvidia/cloud-native \
  --set kataManager.image=k8s-kata-manager \
  --set kataManager.version=v0.2.4 \
  --set ccManager.enabled=true \
  --set ccManager.defaultMode=on \
  --set ccManager.repository=nvcr.io/nvidia/cloud-native \
  --set ccManager.image=k8s-cc-manager \
  --set ccManager.version=v0.2.0 \
  --set sandboxDevicePlugin.repository=ghcr.io/nvidia \
  --set sandboxDevicePlugin.image=nvidia-sandbox-device-plugin \
  --set sandboxDevicePlugin.version=8e76fe81 \
  --set 'sandboxDevicePlugin.env[0].name=P_GPU_ALIAS' \
  --set 'sandboxDevicePlugin.env[0].value=pgpu' \
  --set nfd.enabled=true \
  --set nfd.nodefeaturerules=true

# Wait until nvidia-cc-manager daemonset exists
until kubectl get daemonset nvidia-cc-manager -n gpu-operator &>/dev/null; do
  echo "Waiting for nvidia-cc-manager daemonset to be created..."
  sleep 5
done

if [[ $BLACKWELL == "true" ]]; then
  # Support Blackwell GPUs in nvidia-cc-manager
  current_ids=$(kubectl get daemonset nvidia-cc-manager -n gpu-operator -o jsonpath='{.spec.template.spec.containers[?(@.name=="nvidia-cc-manager")].env[?(@.name=="CC_CAPABLE_DEVICE_IDS")].value}')
  kubectl set env daemonset/nvidia-cc-manager -n gpu-operator -c nvidia-cc-manager CC_CAPABLE_DEVICE_IDS="${current_ids},0x2901"

  # Deploy nvidia-cc-manager to nodes with Blackwell GPUs
  kubectl patch nodefeaturerule nvidia-nfd-nodefeaturerules --type='json' -p='[
    {
      "op": "add",
      "path": "/spec/rules/10",
      "value": {
        "name": "NVIDIA B200",
        "labels": {
          "nvidia.com/gpu.B200": "true",
          "nvidia.com/gpu.family": "blackwell"
        },
        "matchFeatures": [
          {
            "feature": "pci.device",
            "matchExpressions": {
              "device": {"op": "In", "value": ["2901"]},
              "vendor": {"op": "In", "value": ["10de"]}
            }
          }
        ]
      }
    },
    {
      "op": "add",
      "path": "/spec/rules/11/matchAny/0/matchFeatures/0/matchExpressions/nvidia.com~1gpu.family/value/-",
      "value": "blackwell"
    },
    {
      "op": "add",
      "path": "/spec/rules/11/matchAny/1/matchFeatures/0/matchExpressions/nvidia.com~1gpu.family/value/-",
      "value": "blackwell"
    }
  ]'
fi

until [[ "$(
  kubectl get nodes -l nvidia.com/gpu.present -o json |
    jq '.items[0].status.allocatable
        | with_entries(select(.key | startswith("nvidia.com/")))
        | with_entries(select(.value != "0"))'
)" != '{}' ]]; do
  echo 'Waiting for GPU to become available...'
  sleep 5
done
