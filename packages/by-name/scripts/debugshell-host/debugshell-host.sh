#!/usr/bin/env bash
# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "Usage: $0 <namespace> <pod> [args...]" >&2
  exit 1
fi

namespace=$1
podname=$2
shift 2

sandbox=$(crictl --config /dev/null --runtime-endpoint "${CONTAINER_RUNTIME_ENDPOINT:-/run/containerd/containerd.sock}" inspectp |
  jq '.[]' |
  jq "select(.status.metadata.name == \"$podname\")" |
  jq "select(.status.metadata.namespace == \"$namespace\")" |
  jq 'select(.status.state == "SANDBOX_READY")' |
  jq -rs '.[0].status.id')

# URL format: vsock://$CID:1024
cid=$(cat "/run/vc/sbs/$sandbox/persist.json" | jq -r .AgentState.URL | sed -e 's|vsock://||' -e 's|:.*$||')

key=$(mktemp -d)
trap 'rm -rf "$key"' EXIT
ssh-keygen -t ed25519 -f "$key/id_ed25519" -C "" -N ""
ssh -o "ProxyCommand=socat - VSOCK-CONNECT:$cid:22" -o StrictHostKeyChecking=false -i "$key/id_ed25519" root@localhost "$@"
