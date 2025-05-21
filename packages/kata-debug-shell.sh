#!/usr/bin/env bash
# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

if [ "$(id -u)" -ne 0 ]; then
  echo "Please run as root"
  exit 1
fi

if [ -z "${1:-}" ]; then
  echo "Usage: $0 <container_id>"
  exit 1
fi

container_info=$(k3s ctr c info "$1")

sbx_id=$(echo "$container_info" | jq -r '.Spec.annotations."io.kubernetes.cri.sandbox-id"')
runtime_class_name=$(echo "$container_info" | jq -r '.Snapshotter' | cut -c7-)

kata_runtime="/opt/edgeless/${runtime_class_name}/bin/kata-runtime"
config_file=$(ls -1 /opt/edgeless/"${runtime_class_name}"/etc/configuration-*.toml)

${kata_runtime} --config "${config_file}" exec "${sbx_id}"
