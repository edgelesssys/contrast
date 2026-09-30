#!/usr/bin/env bash
# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

since="${1:?usage: collect-host-logs <since>}"
node="${NODE_NAME:?NODE_NAME must be set}"
hostdir="/export/logs/host/$node"
mkdir -p "$hostdir"
echo "Collecting kernel logs (since $since)..." >&2
journalctl --directory=/journal -o short-iso-precise -k -q --since="$since" --no-pager >"$hostdir/kernel.log" 2>/dev/null || true
echo "Collecting k3s logs (since $since)..." >&2
journalctl --directory=/journal -o short-iso-precise -u k3s -q --since="$since" --no-pager >"$hostdir/k3s.log" 2>/dev/null || true
echo "Collecting kubelet logs (since $since)..." >&2
journalctl --directory=/journal -o short-iso-precise -u kubelet -q --since="$since" --no-pager >"$hostdir/kubelet.log" 2>/dev/null || true
echo "Collecting containerd logs (since $since)..." >&2
journalctl --directory=/journal -o short-iso-precise -u containerd -q --since="$since" --no-pager >"$hostdir/containerd.log" 2>/dev/null || true
# k3s runs its own containerd, which logs to a file instead of the journal.
k3s_containerd_dir=/var/lib/rancher/k3s/agent/containerd
if [[ -d $k3s_containerd_dir ]] && since_epoch=$(date -u -d "$since" +%s); then
  # k3s writes containerd's log through lumberjack, which rotates it at 50 MB
  # into containerd-<UTC rotation time>.log and gzips that in the background.
  # The .log is deleted only once the .gz is complete, and the .gz if
  # compression fails, so a backup is read from the .log if it still exists
  # and from the .gz otherwise. Read the backups rotated after $since, oldest
  # first, then the current file.
  stamps=()
  for f in "$k3s_containerd_dir"/containerd-*.log "$k3s_containerd_dir"/containerd-*.log.gz; do
    [[ -e $f ]] || continue
    stamp=${f##*/containerd-}
    stamp=${stamp%.gz}
    stamp=${stamp%.log}
    rotated=$(date -u -d "${stamp:0:10} ${stamp:11:2}:${stamp:14:2}:${stamp:17:2}" +%s) || continue
    if ((rotated >= since_epoch)); then
      stamps+=("$stamp")
    fi
  done
  current=$k3s_containerd_dir/containerd.log
  if ((${#stamps[@]} > 0)) || [[ -s $current ]]; then
    echo "Collecting k3s containerd logs (since $since)..." >&2
    # Skip ahead to the first line whose time="..." field is at or after $since
    # and keep everything from there on.
    {
      if ((${#stamps[@]} > 0)); then
        while read -r stamp; do
          backup=$k3s_containerd_dir/containerd-$stamp.log
          cat "$backup" 2>/dev/null || gzip -dc "$backup.gz" 2>/dev/null || true
        done < <(printf '%s\n' "${stamps[@]}" | sort -u)
      fi
      if [[ -e $current ]]; then
        cat "$current" || true
      fi
    } | gawk -v since="$since_epoch" '
      !found && match($0, /time="([0-9]{4})-([0-9]{2})-([0-9]{2})T([0-9]{2}):([0-9]{2}):([0-9]{2})[.0-9]*(Z|([+-])([0-9]{2}):([0-9]{2}))"/, t) {
        ts = mktime(t[1] " " t[2] " " t[3] " " t[4] " " t[5] " " t[6], 1)
        if (t[7] != "Z") {
          off = (t[9] * 3600 + t[10] * 60) * (t[8] == "-" ? -1 : 1)
          ts -= off
        }
        if (ts >= since) found = 1
      }
      found
    ' >"$hostdir/containerd.log" || true
  fi
fi
echo "Collecting kata logs (since $since)..." >&2
journalctl --directory=/journal -o short-iso-precise -t kata -q --since="$since" --no-pager >"$hostdir/kata.log" 2>/dev/null || true
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
