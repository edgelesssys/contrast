#!/usr/bin/env bash
# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

set -euo pipefail

knownNamespaces=(
  "default"
  "maintenance-cleanup"
  "maintenance-nix-gc"
  "maintenance-namespace-cleanup"
  "gpu-operator"
  "kube-node-lease"
  "kube-public"
  "kube-system"
  "longhorn-system"
)

kubectl get namespaces --no-headers | while read -r ns _; do
  if [[ " ${knownNamespaces[*]} " == *" $ns "* ]]; then
    echo "Skipping known namespace: $ns"
    continue
  fi

  timestamp=$(kubectl get namespace "$ns" -o jsonpath='{.metadata.creationTimestamp}')
  time=$(date -D '%Y-%m-%dT%H:%M:%SZ' -d "$timestamp" +%s)

  if [[ $time -lt "$(($(date +%s) - 3600))" ]]; then
    echo "Deleting namespace: $ns"
    kubectl delete namespace "$ns"
  else
    echo "Skipping namespace: $ns (created within the last hour)"
  fi
done
