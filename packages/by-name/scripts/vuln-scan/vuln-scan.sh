#!/usr/bin/env bash
# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

sbom="${1:?usage: vuln-scan <sbom.cdx.json> [sarif-output-dir]}"
sarifdir="${2:-}"

sbom=$(realpath -- "$sbom")
[[ -n $sarifdir ]] && mkdir -p "$sarifdir"

exitcode=0
workdir=$(mktemp -d)

# osv derives its result fingerprints, which GitHub code scanning uses for alert identity, from the scanned file's PATH.
# Therefore, scan from a fixed, content-hash-free path.
osvdir="/tmp/vuln-scan"
mkdir -p "$osvdir"
base="${sbom##*/}"
relabeled="$osvdir/${base#*-}"
trap 'rm -rf "$workdir"; rm -f "$relabeled"' EXIT

# bombon emits CycloneDX 1.7, which osv-scanner cannot parse yet. Can be dropped when osv-scanner supports CDX 1.7.
jq '.specVersion = "1.6"' "$sbom" >"$relabeled"

if [[ -n $sarifdir ]]; then
  osv-scanner scan source \
    --config="$OSV_CONFIG" \
    --format=sarif --output="$sarifdir/osv.sarif" \
    -L "$relabeled" || exitcode=$?
fi
osv-scanner scan source \
  --config="$OSV_CONFIG" \
  --format=table \
  -L "$relabeled" || exitcode=$?

vulnix_json="$workdir/vulnix.json"
vulnix --closure --json --whitelist "$VULNIX_WHITELIST" "$sbom" >"$vulnix_json" || exitcode=$?
jq -r 'if length == 0 then "vulnix: no advisories"
       else (.[] | "vulnix: \(.name) affected by \(.affected_by | join(", "))") end' \
  "$vulnix_json"
if [[ -n $sarifdir ]]; then
  jq --arg sbom "file://$relabeled" -f "$VULNIX_SARIF_JQ" "$vulnix_json" >"$workdir/vulnix.sarif" &&
    mv "$workdir/vulnix.sarif" "$sarifdir/vulnix.sarif"
fi

exit "$exitcode"
