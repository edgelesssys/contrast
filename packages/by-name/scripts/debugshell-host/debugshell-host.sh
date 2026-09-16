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
  jq -r -f "$JQ_FIND_SANDBOX_ID" --arg namespace "$namespace" --arg podname "$podname")

# URL format: vsock://$CID:1024
cid=$(cat "/run/vc/sbs/$sandbox/persist.json" | jq -r .AgentState.URL | sed -e 's|vsock://||' -e 's|:.*$||')

if ! echo -n "$cid" | grep -qE "^[0-9]+$"; then
  # The above pipeline succeeded, so running only parts of it should succeed, too.
  echo "Could not parse numeric CID from agent URL $(jq -r .AgentState.URL <"/run/vc/sbs/$sandbox/persist.json")" >&2
  exit 2
fi

key=$(mktemp -d)
trap 'rm -rf "$key"' EXIT
ssh-keygen -t ed25519 -f "$key/id_ed25519" -C "" -N "" >&2
ssh \
  -o "ProxyCommand=socat - VSOCK-CONNECT:$cid:22" \
  -o StrictHostKeyChecking=no \
  -o UserKnownHostsFile=/dev/null \
  -i "$key/id_ed25519" \
  root@localhost "$@"
