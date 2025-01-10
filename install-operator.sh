#!/usr/bin/env bash
# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

# See https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/latest/gpu-operator-confidential-containers.html

helm install --wait --generate-name \
  -n gpu-operator --create-namespace \
  nvidia/gpu-operator \
  --version=v24.9.0 \
  --set sandboxWorkloads.enabled=true \
  --set sandboxWorkloads.defaultWorkload='vm-passthrough' \
  --set ccManager.enabled=true \
  --set nfd.nodefeaturerules=true \
  --set vfioManager.enabled=true \
  --set kataManager.enabled=true

kubectl label node discovery nvidia.com/cc.mode=on --overwrite
