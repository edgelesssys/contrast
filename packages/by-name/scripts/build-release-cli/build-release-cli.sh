#!/usr/bin/env bash
# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

# Build the release CLI and SBOM for <system> and write it to <out>. Fail
# unless the CLI matches the SBOM.
#
# The image replacements must already be written to
# cli/cmd/assets/image-replacements.txt, as they are embedded into the CLI.
# <system> must match the host system of the CLI build.

set -euo pipefail

usage() {
  echo "usage: build-release-cli --system <system> --out <dir>" >&2
  exit 2
}

while [[ $# -gt 0 ]]; do
  case "$1" in
  --system) system="${2:-}" ;;
  --out) out="${2:-}" ;;
  *) usage ;;
  esac
  shift 2 || usage
done
[[ -n ${system-} && -n ${out-} ]] || usage

# Build the CLI for the given system and write it to <out>.
cli=$(nix build -L --no-link --print-out-paths .#base.contrast.cli-release)
mkdir -p "$out"
install -m 0755 "$cli/bin/contrast" "$out/contrast-$system"

# Build the corresponding SBOM for the given system and write it to <out>.
sbom=$(finalize-sbom cli --artifact "$out/contrast-$system")
name=$(jq -r '.metadata.component.name' <<<"$sbom")
if [[ $name != "contrast-cli-$system" ]]; then
  echo "SBOM identifies itself as '$name', expected 'contrast-cli-$system'" >&2
  exit 1
fi
printf '%s\n' "$sbom" >"$out/$name.cdx.json"

# The SBOM has to describe the Nix output we just built, not some other one.
hash=$(nix-hash --type sha256 --base32 "$cli")
if ! jq -e --arg ref "${cli##*/}" --arg hash "sha256:$hash" '
  .components | any(
    .["bom-ref"] == $ref and
    any(.properties[]?; .name == "nix:narinfo:nar_hash" and .value == $hash)
  )
' "$out/$name.cdx.json" >/dev/null; then
  echo "CLI $cli doesn't match SBOM $name" >&2
  exit 1
fi
