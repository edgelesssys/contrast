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

# vulnix downloads the NVD feeds from nvd.nist.gov on every run. Reuse a persistent cache dir when one is provided
# Additionally, retry transient failures. Exit code 2 means "vulnerabilities found".
vulnix_json="$workdir/vulnix.json"
vulnix_args=(--closure --json --whitelist "$VULNIX_WHITELIST")
[[ -n ${VULNIX_CACHE_DIR:-} ]] && vulnix_args+=(--cache-dir "$VULNIX_CACHE_DIR")

vx_rc=0
for attempt in 1 2 3; do
  vx_rc=0
  vulnix "${vulnix_args[@]}" "$sbom" >"$vulnix_json" || vx_rc=$?
  [[ $vx_rc -eq 0 || $vx_rc -eq 2 ]] && break
  echo "vulnix: exit $vx_rc (NVD download/network issue?); retry $attempt/3 after backoff" >&2
  sleep $((attempt * 20))
done

if [[ $vx_rc -eq 0 || $vx_rc -eq 2 ]]; then
  jq -r 'if length == 0 then "vulnix: no advisories"
         else (.[] | "vulnix: \(.name) affected by \(.affected_by | join(", "))") end' \
    "$vulnix_json"
  if [[ -n $sarifdir ]]; then
    jq --arg sbom "file://$relabeled" -f "$VULNIX_SARIF_JQ" "$vulnix_json" >"$workdir/vulnix.sarif" &&
      mv "$workdir/vulnix.sarif" "$sarifdir/vulnix.sarif"
  fi
  [[ $vx_rc -eq 0 ]] || exitcode=$vx_rc
else
  echo "vulnix: could not reach NVD after 3 attempts (exit $vx_rc)" >&2
  exitcode=$vx_rc
fi

exit "$exitcode"
