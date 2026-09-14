#!/usr/bin/env bash
# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

while [ $# -gt 0 ]; do
  case "$1" in
  --image-replacements)
    imageReplacements="$2"
    ;;
  --namespace)
    ns="$2"
    ;;
  --runtime-class)
    runtimeClass="$2"
    ;;
  --rules)
    rules="$2"
    ;;
  --settings)
    settings="$2"
    ;;
  --yaml)
    yaml="$2"
    ;;
  --output)
    output="$2"
    ;;
  *)
    echo "Unknown option: $1"
    exit 1
    ;;
  esac
  shift 2
done

dir=$(mktemp -d)
trap 'rm -rf "$dir"' EXIT

cp "$yaml" "$dir/pod.yml"

cat <<EOF >"$dir/initdata.toml"
version = "0.1.0"
algorithm = "sha256"
[data]
"contrast.insecure-debug" = "true"
"agent.toml" = "log_level = \"debug\""
EOF

debugshell=$(grep '^ghcr.io/edgelesssys/contrast/debugshell:latest=' "$imageReplacements" | tail -1 | cut -d= -f2-)

ns=$ns debugshell=$debugshell yq \
  '.metadata.namespace = env(ns) | .spec.containers[0].image = env(debugshell)' \
  "$DEBUGGER_YAML" >"$dir/debugger.yml"

rc=$runtimeClass ns=$ns yq -i \
  '.metadata.namespace = env(ns) | .spec.runtimeClassName = env(rc)' \
  "$dir/pod.yml"

genpolicy \
  --runtime-class-names="$runtimeClass" \
  --rego-rules-path="$rules" \
  --json-settings-path="$settings" \
  --layers-cache-file-path="$dir/layers-cache.json" \
  --initdata-path="$dir/initdata.toml" \
  --yaml-file="$dir/pod.yml"

kubectl apply -f "$dir/debugger.yml"
kubectl apply -f "$dir/pod.yml"

kubectl wait --namespace "$ns" --for=condition=Ready pod/debugger --timeout=60s
kubectl wait --namespace "$ns" --for=condition=Ready pod/test --timeout=180s

kubectl exec --namespace "$ns" pod/debugger -- \
  debugshell-host "$ns" test \
  cat /tmp/policy.jsonl 2>/dev/null >"$output"
