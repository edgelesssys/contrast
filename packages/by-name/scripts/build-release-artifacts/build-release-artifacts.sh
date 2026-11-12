#!/usr/bin/env bash
# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

usage() {
  echo "usage: build-release-artifacts --image-replacements <file> --out <dir>" >&2
  exit 2
}

while [[ $# -gt 0 ]]; do
  case "$1" in
  --image-replacements) replacements="${2:-}" ;;
  --out) out="${2:-}" ;;
  *) usage ;;
  esac
  shift 2 || usage
done
[[ -n ${replacements-} && -n ${out-} ]] || usage

mkdir -p "$out"

# Create Coordinator resource
resourcegen \
  --image-replacements "$replacements" \
  --add-load-balancers \
  coordinator >"$out/coordinator.yml"

# Create runtime resources for each platform
platforms=(
  "metal-qemu-snp-gpu"
  "metal-qemu-snp"
  "metal-qemu-tdx"
  "metal-qemu-tdx-gpu"
)
for platform in "${platforms[@]}"; do
  resourcegen \
    --image-replacements "$replacements" \
    --namespace contrast-system \
    --add-namespace-object \
    --platform "$platform" \
    runtime >"$out/runtime-$platform.yml"
done

# Create demo resources
resourcegen \
  --image-replacements "$replacements" \
  --add-load-balancers emojivoto-sm-ingress >"$out/emojivoto-demo.yml"
resourcegen \
  --image-replacements "$replacements" \
  --add-load-balancers mysql >"$out/mysql-demo.yml"
resourcegen \
  --image-replacements "$replacements" \
  --add-load-balancers vault >"$out/vault-demo.yml"

# Create node installer target configs
distros=(
  k3s
  rke2
)
for distro in "${distros[@]}"; do
  resourcegen \
    --node-installer-target-conf-type "$distro" \
    --namespace contrast-system \
    --add-namespace-object \
    node-installer-target-conf >"$out/node-installer-target-config-${distro}.yml"
done

# Create runtime SBOMs
for category in runtimeConfidential runtimeNonConfidential; do
  sbom=$(finalize-sbom "$category")
  id=$(jq -r '.metadata.component.name' <<<"$sbom")
  printf '%s\n' "$sbom" >"$out/${id}.cdx.json"
done
