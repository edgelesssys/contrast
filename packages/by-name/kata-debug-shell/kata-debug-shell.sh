#!/usr/bin/env bash
# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

if [[ -z ${1:-} || ${1:-} == "--help" ]]; then
  cat <<EOF >&2
Usage: $0 <namespace/pod-name>

Utility script to get a debug shell in a Kata Containers sandbox VM.

The script collects the following information from Kubernetes/containerd:
  - runtimeClass of the pod
  - sandbox ID of the pod
With this, it calls the 'kata-runtime' binary installed as part of the runtimeClass,
using the config file from the same runtimeClass and the sandbox ID to get a shell.
EOF
  exit 1
fi

readonly ctr_namespace="k8s.io"

die() {
  echo "Error: $1" >&2
  exit 1
}

if [[ "$(id -u)" -ne 0 ]]; then
  die "Please run as root"
fi

if command -v k3s &>/dev/null; then
  ctr_cmd="k3s ctr"
elif command -v ctr &>/dev/null; then
  ctr_cmd="ctr"
else
  die "Neither 'k3s ctr' nor 'ctr' command found. Please install one of them."
fi

pod_namespace="${1%/*}"
pod_name="${1##*/}"

if [[ $1 != */* ]] || [[ -z $pod_namespace ]] || [[ -z $pod_name ]]; then
  die "Could not parse namespace or podname from input '$1'."
fi

runtime_class=$(
  kubectl get pod "${pod_name}" \
    -n "${pod_namespace}" \
    -o=jsonpath='{.spec.runtimeClassName}'
) || die "Failed to get runtime class for pod '${pod_name}' in namespace '${pod_namespace}'."
echo "Found runtime class ${runtime_class} for pod '${pod_name}' in namespace '${pod_namespace}'." >&2

filters=(
  "labels.\"io.kubernetes.pod.namespace\"==${pod_namespace}"
  "labels.\"io.kubernetes.pod.name\"==${pod_name}"
  'labels."io.cri-containerd.kind"==sandbox'
)
filter_str=$(
  IFS=,
  echo "${filters[*]}"
)

sandbox_id=$(
  ${ctr_cmd} -n "${ctr_namespace}" c ls "${filter_str}" |
    tail -n1 |
    cut -d' ' -f1
) || die "Failed to find sandbox id for pod '${pod_name}' in namespace '${pod_namespace}'."
echo "Found sandbox id ${sandbox_id}" >&2

kata_runtime="/opt/edgeless/${runtime_class}/bin/kata-runtime"
kata_config_file=$(ls -1 /opt/edgeless/"${runtime_class}"/etc/configuration-*.toml)

${kata_runtime} --config "${kata_config_file}" exec "${sandbox_id}"
