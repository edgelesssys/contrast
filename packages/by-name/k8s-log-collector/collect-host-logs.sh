#!/usr/bin/env bash
# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

since="${1:?usage: collect-host-logs <since>}"
node="${NODE_NAME:?NODE_NAME must be set}"
hostdir="/export/logs/host/$node"
mkdir -p "$hostdir"
echo "Collecting kernel logs (since $since)..." >&2
journalctl --directory=/journal -k -q --since="$since" --no-pager >"$hostdir/kernel.log" 2>/dev/null || true
echo "Collecting k3s logs (since $since)..." >&2
journalctl --directory=/journal -u k3s -q --since="$since" --no-pager >"$hostdir/k3s.log" 2>/dev/null || true
echo "Collecting kubelet logs (since $since)..." >&2
journalctl --directory=/journal -u kubelet -q --since="$since" --no-pager >"$hostdir/kubelet.log" 2>/dev/null || true
echo "Collecting containerd logs (since $since)..." >&2
journalctl --directory=/journal -u containerd -q --since="$since" --no-pager >"$hostdir/containerd.log" 2>/dev/null || true
echo "Collecting kata logs (since $since)..." >&2
journalctl --directory=/journal -t kata -q --since="$since" --no-pager >"$hostdir/kata.log" 2>/dev/null || true
# Remove empty log files (services not running on this node).
for f in "$hostdir"/*.log; do
  [[ -s $f ]] || rm -f "$f"
done
echo "Collecting pod-sandbox metadata..." >&2
mkdir -p "/export/logs/metadata/$node"
for sock in /run/k3s/containerd/containerd.sock /run/containerd/containerd.sock; do
  if [[ -S $sock ]]; then
    CONTAINER_RUNTIME_ENDPOINT="unix://$sock" crictl pods -o json 2>/dev/null |
      jq -r --arg ns "${POD_NAMESPACE:-}" \
        '.items[] | select(.metadata.namespace == $ns and .runtimeHandler != "" and .runtimeHandler != null) | "\(.metadata.name)\t\(.id)"' \
        >"/export/logs/metadata/$node/sandbox-map.txt"
    break
  fi
done
echo "Host log collection complete." >&2
