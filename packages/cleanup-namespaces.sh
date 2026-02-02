#!/usr/bin/env bash
# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

set -euo pipefail

knownNamespaces=(
  # keep-sorted start
  "continuum-*"
  "csi-system"
  "default"
  "gpu-operator"
  "kube-node-lease"
  "kube-public"
  "kube-system"
  "longhorn-system"
  "maintenance-cleanup"
  "maintenance-namespace-cleanup"
  "maintenance-nix-gc"
  "privatemode-*"
  # keep-sorted end
)

kubectl get namespaces --no-headers | while read -r ns _; do
  skip=false
  for pattern in "${knownNamespaces[@]}"; do
    # shellcheck disable=SC2053 # We want glob matching here
    if [[ $ns == $pattern ]]; then
      echo "Skipping known namespace: $ns"
      skip=true
      break
    fi
  done
  if [[ $skip == true ]]; then
    continue
  fi

  timestamp=$(kubectl get namespace "$ns" -o jsonpath='{.metadata.creationTimestamp}')
  time=$(date -D '%Y-%m-%dT%H:%M:%SZ' -d "$timestamp" +%s)

  if [[ $time -lt "$(($(date +%s) - 3600))" ]]; then
    echo "Deleting namespace: $ns"
    sync_ticket=$(kubectl get namespace "$ns" -o jsonpath='{.metadata.labels.contrast\.edgeless\.systems/sync-ticket}')
    kubectl delete namespace "$ns"
    if [[ -n $sync_ticket ]]; then
      sync_ip=$(kubectl get svc -n default sync -o jsonpath='{.spec.clusterIP}')
      sync_uuid=$(kubectl get configmap -n default sync-server-fifo -o jsonpath='{.data.uuid}')
      curl -fsSL "$sync_ip:8080/fifo/$sync_uuid/done/$sync_ticket"
    fi
  else
    echo "Skipping namespace: $ns (created within the last hour)"
  fi
done
