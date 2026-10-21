#!/usr/bin/env bash
# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

usage() {
  echo "usage: finalize-sbom <full|cli|runtimeConfidential|runtimeNonConfidential> [output-file] [--artifact <released-file>]" >&2
  exit 2
}

artifact=""
positional=()
while [[ $# -gt 0 ]]; do
  case "$1" in
  --artifact)
    [[ $# -ge 2 ]] || usage
    artifact="$2"
    shift 2
    ;;
  *)
    positional+=("$1")
    shift
    ;;
  esac
done
set -- ${positional[@]+"${positional[@]}"}

[[ $# -ge 1 && $# -le 2 ]] || usage

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

artifact_uri=""
artifact_sha256=""
artifact_sha512=""
if [[ -n $artifact ]]; then
  artifact_sha256=$(sha256sum "$artifact" | cut -d' ' -f1)
  artifact_sha512=$(sha512sum "$artifact" | cut -d' ' -f1)
  artifact_uri="https://github.com/edgelesssys/contrast/releases/download/v${version}/$(basename "$artifact")"
fi

jq \
  --arg ts "$timestamp" \
  --arg uri "$sbom_uri" \
  --arg artifactUri "$artifact_uri" \
  --arg sha256 "$artifact_sha256" \
  --arg sha512 "$artifact_sha512" '
    [ { alg: "SHA-256", content: $sha256 }, { alg: "SHA-512", content: $sha512 } ] as $digests
    | .metadata.timestamp = $ts
    | .externalReferences = ((.externalReferences // []) + [ { type: "bom", url: $uri } ])
    | if $artifactUri == "" then . else
        .metadata.component.hashes = $digests
        | .metadata.component.externalReferences =
            ((.metadata.component.externalReferences // [])
             + [ { type: "distribution", url: $artifactUri, hashes: $digests } ])
      end
  ' "$sbom" >"$outfile"
