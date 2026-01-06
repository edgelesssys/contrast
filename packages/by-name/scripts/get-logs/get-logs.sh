#!/usr/bin/env bash
# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

retry() {
  local retries=5
  local count=0
  local delay=5
  until "$@"; do
    exit_code=$?
    count=$((count + 1))
    if [ "$count" -lt "$retries" ]; then
      echo "Command failed. Attempt $count/$retries. Retrying in $delay seconds..." >&2
      sleep $delay
    else
      echo "Command failed after $retries attempts. Exiting." >&2
      return $exit_code
    fi
  done
}

if [[ $# -lt 2 ]]; then
  echo "Usage: get-logs [start | download] namespaceFile"
  exit 1
fi

case $1 in
start)
  while ! [[ -s $2 ]]; do
    sleep 1
  done
  # Check if namespace file exists
  # Since no file exists if no test is run, exit gracefully
  if [[ ! -f $2 ]]; then
    echo "Namespace file $2 does not exist" >&2
    exit 0
  fi
  while IFS= read -r namespace; do
    cp ./packages/log-collector.yaml ./workspace/log-collector.yaml
    retry kubectl apply -n "$namespace" -f ./workspace/log-collector.yaml 1>/dev/null 2>/dev/null
  done < <(tail -n +1 -f "$2")
  ;;
download)
  if [[ ! -f $2 ]]; then
    echo "Namespace file $2 does not exist" >&2
    exit 0
  fi
  while read -r namespace; do
    pod="$(kubectl get pods -o name -n "$namespace" | grep log-collector | cut -c 5-)"
    mkdir -p "./workspace/logs/$namespace"
    retry kubectl wait --for=condition=Ready -n "$namespace" "pod/$pod"
    retry kubectl exec -n "$namespace" "$pod" -- /bin/bash -c "rm -f /exported-logs.tar.gz; cp -r /export /export-no-stream; tar zcvf /exported-logs.tar.gz /export-no-stream; rm -rf /export-no-stream"
    retry kubectl cp -n "$namespace" "$pod:/exported-logs.tar.gz" ./workspace/logs/exported-logs.tar.gz
    tar xzvf ./workspace/logs/exported-logs.tar.gz --directory "./workspace/logs/$namespace"
    rm ./workspace/logs/exported-logs.tar.gz
    retry kubectl events -n "$namespace" -o yaml >"./workspace/logs/$namespace/export-no-stream/logs/k8s-events.yaml"
  done <<<"$(cat "$2")"
  ;;
*)
  echo "Unknown option $1"
  echo "Usage: get-logs [start | download] namespaceFile"
  exit 1
  ;;
esac
