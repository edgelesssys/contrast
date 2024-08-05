#!/usr/bin/env bash

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

for image in "${pauseImages[@]}"; do
  ctr "${ctrOpts[@]}" content fetch "${image}"
done
