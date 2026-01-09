#!/usr/bin/env bash
# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail
set -m

retry() {
  local retries=5
  local count=0
  local delay=5
  until "$@"; do
    exit_code=$?
    count=$((count + 1))
    if [[ $count -lt $retries ]]; then
      echo "Command failed. Attempt $count/$retries. Retrying in $delay seconds..." >&2
      sleep $delay
    else
      echo "Command failed after $retries attempts. Exiting." >&2
      return $exit_code
    fi
  done
}

deploy_collectors() {
  local namespace_file="$1"
  # shellcheck disable=SC2329
  cleanup() {
    set +x
    trap - INT TERM EXIT
    kill -- -$$ 2>/dev/null || true
  }
  trap cleanup INT TERM EXIT
  tail -n +1 -f "$namespace_file" |
    while IFS= read -r namespace; do
      cp ./packages/log-collector.yaml ./workspace/log-collector.yaml
      echo "Starting log collector in namespace $namespace" >&2
      retry kubectl apply -n "$namespace" -f ./workspace/log-collector.yaml
    done
}

kill_deploy_collectors() {
  local namespace_file="$1"
  local deploy_pid="$2"
  local dir base
  dir="$(dirname -- "$namespace_file")"
  base="$(basename -- "$namespace_file")"
  while :; do
    inotifywait -q -e delete,moved_from --format '%f' "$dir" |
      grep -qx "$base" && break
  done
  echo "Namespace file $namespace_file deleted, terminating log collector deployment..." >&2
  kill -- -"$deploy_pid" 2>/dev/null || true
  wait "$deploy_pid" 2>/dev/null || true
}

if [[ $# -lt 2 ]]; then
  echo "Usage: get-logs [start | download] namespaceFile"
  exit 1
fi

case $1 in
start)
  namespace_file="$2"
  while ! [[ -s $namespace_file ]]; do
    sleep 1
  done
  deploy_collectors "$namespace_file" &
  deploy_pid=$!
  kill_deploy_collectors "$namespace_file" "$deploy_pid" &
  ;;
download)
  if [[ ! -f $2 ]]; then
    echo "Namespace file $2 does not exist" >&2
    exit 0
  fi
  while read -r namespace; do
    pod="$(kubectl get pods -o name -n "$namespace" | grep log-collector | cut -c 5-)"
    echo "Collecting logs from namespace $namespace, pod $pod" >&2
    mkdir -p "./workspace/logs/$namespace"
    retry kubectl wait --for=condition=Ready -n "$namespace" "pod/$pod"
    echo "Pod $pod is ready" >&2
    retry kubectl exec -n "$namespace" "$pod" -- /bin/bash -c "rm -f /exported-logs.tar.gz; cp -r /export /export-no-stream; tar zcvf /exported-logs.tar.gz /export-no-stream; rm -rf /export-no-stream"
    retry kubectl cp -n "$namespace" "$pod:/exported-logs.tar.gz" ./workspace/logs/exported-logs.tar.gz
    echo "Downloaded logs tarball for namespace $namespace, extracting..." >&2
    tar xzvf ./workspace/logs/exported-logs.tar.gz --directory "./workspace/logs/$namespace"
    rm ./workspace/logs/exported-logs.tar.gz
    echo "Collecting Kubernetes events for namespace $namespace" >&2
    retry kubectl events -n "$namespace" -o yaml >"./workspace/logs/$namespace/export-no-stream/logs/k8s-events.yaml"
    echo "Logs for namespace $namespace collected successfully" >&2
  done <<<"$(cat "$2")"
  ;;
*)
  echo "Unknown option $1"
  echo "Usage: get-logs [start | download] namespaceFile"
  exit 1
  ;;
esac
