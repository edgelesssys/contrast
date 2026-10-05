#!/usr/bin/env bash
# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

while [ $# -gt 0 ]; do
  case "$1" in
  --image-replacements)
    imageReplacements="$2"
    ;;
  --target-conf-type)
    targetConfType="$2"
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
  --testdata)
    testdata="$2"
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

# Use custom initdata to enable the SSH server via contrast.insecure-debug
# and set agent log level so the policy requests are logged to /tmp/policy.jsonl.
cat <<EOF >"$dir/initdata.toml"
version = "0.1.0"
algorithm = "sha256"
[data]
"contrast.insecure-debug" = "true"
"agent.toml" = "log_level = \"debug\""
EOF

debugshell=$(grep '^ghcr.io/edgelesssys/contrast/debugshell:latest=' "$imageReplacements" | tail -1 | cut -d= -f2-)

containerdPath="/run/containerd/containerd.sock"
if [[ ${targetConfType} == "k3s" ]]; then
  containerdPath="/var/run/k3s/containerd/containerd.sock"
fi
sed "s|@@REPLACE_CTR_PATH@@|$containerdPath|" "$DEBUGGER_YAML" >"$dir/debugger.yml"

ns=$ns debugshell=$debugshell yq -i \
  '.metadata.namespace = env(ns) | .spec.containers[0].image = env(debugshell)' \
  "$dir/debugger.yml"

# Resources are stored under testdata/<test-name>/resource.yml.
# Iterate over each subdir and deploy the generated resource.
for subdir in "$testdata"/*; do
  sed "s/contrast-cc/$runtimeClass/" "$subdir/resource.yml" >"$dir/resource.yml"
  ns=$ns yq -i '.metadata.namespace = env(ns)' "$dir/resource.yml"

  genpolicy \
    --runtime-class-names="$runtimeClass" \
    --rego-rules-path="$rules" \
    --json-settings-path="$settings" \
    --layers-cache-file-path="$dir/layers-cache.json" \
    --initdata-path="$dir/initdata.toml" \
    --yaml-file="$dir/resource.yml"

  kubectl apply -f "$dir/resource.yml"
done

kubectl apply -f "$dir/debugger.yml"
kubectl wait --namespace "$ns" --for=condition=Ready pod/debugger --timeout=60s

# Extract the policy logs for each resource and write the normalized output
# directly to the source tree under testdata/<test-name>/base.json.
for subdir in "$testdata"/*; do
  pod=$(kubectl get pods -n "$ns" -l app.kubernetes.io/name=test-"$(basename "$subdir")" -o jsonpath='{.items[0].metadata.name}')
  kubectl wait --namespace "$ns" --for=condition=Ready "pod/$pod" --timeout=180s
  kubectl exec --namespace "$ns" pod/debugger -- \
    debugshell-host "$ns" "$pod" \
    cat /tmp/policy.jsonl 2>/dev/null >"$dir/policy.jsonl"

  normalize-agent-rpcs "$dir/policy.jsonl" >"$subdir/base.json"
done
