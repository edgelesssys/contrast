#!/usr/bin/env bash
# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

# Script to cleanup messed up containerd/snapshotter state on bare metal.
# Copy to host and execute.

set -euo pipefail

ctrOpts=(
  --address /run/k3s/containerd/containerd.sock
  --namespace k8s.io
)

declare -a pauseImages

while read -r image; do
  ctr "${ctrOpts[@]}" image rm "${image}"
  if [[ $image =~ pause ]]; then
    pauseImages+=("${image}")
  fi
done < <(
  ctr "${ctrOpts[@]}" image list |
    tail -n +2 |
    cut -d' ' -f1
)

while read -r content; do
  ctr "${ctrOpts[@]}" content rm "${content}"
done < <(
  ctr "${ctrOpts[@]}" content list |
    tail -n +2 |
    cut -d$'\t' -f1
)

for image in "${pauseImages[@]}"; do
  ctr "${ctrOpts[@]}" content fetch "${image}"
done

if ctr "${ctrOpts[@]}" image check | grep "incomplete"; then
  echo "Incomplete images detected"
  exit 1
fi
