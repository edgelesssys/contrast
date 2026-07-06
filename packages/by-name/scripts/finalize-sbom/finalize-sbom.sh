#!/usr/bin/env bash
# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

if [[ $# -lt 1 || $# -gt 2 ]]; then
  echo "usage: finalize-sbom <full|cli|runtimeConfidential|runtimeNonConfidential> [output-file]" >&2
  exit 2
fi

category="$1"
outfile="${2:-/dev/stdout}"

case "$category" in
full) attr="sbom" ;;
cli) attr="sbom.cli" ;;
runtimeConfidential) attr="sbom.runtimeConfidential" ;;
runtimeNonConfidential) attr="sbom.runtimeNonConfidential" ;;
*)
  echo "finalize-sbom: unknown category '$category'" >&2
  exit 2
  ;;
esac

sbom=$(nix build --no-link --print-out-paths ".#base.${attr}")

id=$(jq -r '.metadata.component.name' "$sbom")
version=$(jq -r '.metadata.component.version' "$sbom")
timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)
sbom_uri="https://github.com/edgelesssys/contrast/releases/download/v${version}/${id}.cdx.json"

jq \
  --arg ts "$timestamp" \
  --arg uri "$sbom_uri" '
    .metadata.timestamp = $ts
    | .externalReferences = ((.externalReferences // []) + [ { type: "bom", url: $uri } ])
  ' "$sbom" >"$outfile"
