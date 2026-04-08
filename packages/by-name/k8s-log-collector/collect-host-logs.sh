#!/usr/bin/env bash
# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

since="${1:?usage: collect-host-logs <since>}"
mkdir -p /export/logs/host
echo "Collecting kernel logs (since $since)..." >&2
journalctl --directory=/journal -k --since="$since" --no-pager >/export/logs/host/kernel.log 2>/dev/null || true
echo "Collecting k3s logs (since $since)..." >&2
journalctl --directory=/journal -u k3s --since="$since" --no-pager >/export/logs/host/k3s.log 2>/dev/null || true
echo "Collecting kubelet logs (since $since)..." >&2
journalctl --directory=/journal -u kubelet --since="$since" --no-pager >/export/logs/host/kubelet.log 2>/dev/null || true
echo "Collecting containerd logs (since $since)..." >&2
journalctl --directory=/journal -u containerd --since="$since" --no-pager >/export/logs/host/containerd.log 2>/dev/null || true
echo "Collecting kata logs (since $since)..." >&2
journalctl --directory=/journal -t kata --since="$since" --no-pager >/export/logs/host/kata.log 2>/dev/null || true
echo "Collecting pod-sandbox metadata..." >&2
mkdir -p /export/logs/metadata
for sock in /run/k3s/containerd/containerd.sock /run/containerd/containerd.sock; do
  if [[ -S $sock ]]; then
    CONTAINER_RUNTIME_ENDPOINT="unix://$sock" crictl pods -o json 2>/dev/null |
      jq -r --arg ns "${POD_NAMESPACE:-}" \
        '.items[] | select(.metadata.namespace == $ns and .runtimeHandler != "" and .runtimeHandler != null) | "\(.metadata.name)\t\(.id)"' \
        >/export/logs/metadata/sandbox-map.txt
    break
  fi
done
echo "Host log collection complete." >&2
