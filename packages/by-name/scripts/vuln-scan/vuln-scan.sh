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
scanned="$osvdir/${base#*-}"
trap 'rm -rf "$workdir"; rm -f "$scanned"' EXIT

cat "$sbom" >"$scanned"

if [[ -n $sarifdir ]]; then
  osv-scanner scan source \
    --config="$OSV_CONFIG" \
    --format=sarif --output="$sarifdir/osv.sarif" \
    -L "$scanned" || exitcode=$?
fi
osv_json="$workdir/osv.json"
osv-scanner scan source \
  --config="$OSV_CONFIG" \
  --format=json --output="$osv_json" \
  -L "$scanned" || exitcode=$?
# Exit code 1 means "vulnerabilities found", other codes are errors that may leave no results behind.
if [[ -s $osv_json ]]; then
  jq --arg tool osv -f "$NORMALIZE_JQ" "$osv_json" >"$workdir/findings-osv.json"
else
  echo "osv-scanner: no results (exit $exitcode)" >&2
fi

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
  jq --arg tool vulnix -f "$NORMALIZE_JQ" "$vulnix_json" >"$workdir/findings-vulnix.json"
  if [[ -n $sarifdir ]]; then
    jq --arg sbom "file://$scanned" -f "$VULNIX_SARIF_JQ" "$vulnix_json" >"$workdir/vulnix.sarif" &&
      mv "$workdir/vulnix.sarif" "$sarifdir/vulnix.sarif"
  fi
  [[ $vx_rc -eq 0 ]] || exitcode=$vx_rc
else
  echo "vulnix: could not reach NVD after 3 attempts (exit $vx_rc)" >&2
  exitcode=$vx_rc
fi

shopt -s nullglob
findings=("$workdir"/findings-*.json)
if [[ ${#findings[@]} -gt 0 ]]; then
  jq -s add "${findings[@]}" >"$workdir/findings.json"
  # Attribute findings to the packages that vendor them, from the per-package SBOMs in VENDORED_SBOMS.
  vendored=()
  [[ -n ${VENDORED_SBOMS:-} ]] && vendored=("$VENDORED_SBOMS"/*.cdx.json)
  if [[ ${#vendored[@]} -gt 0 ]]; then
    jq -n --slurpfile findings "$workdir/findings.json" -f "$ATTRIBUTE_JQ" "${vendored[@]}" >"$workdir/attributed.json"
    mv "$workdir/attributed.json" "$workdir/findings.json"
  fi
  echo
  jq -r --arg format log --arg title "" -f "$RENDER_JQ" "$workdir/findings.json"
  if [[ -n ${GITHUB_STEP_SUMMARY:-} ]]; then
    jq -r --arg format markdown --arg title "${base#*-}" -f "$RENDER_JQ" \
      "$workdir/findings.json" >>"$GITHUB_STEP_SUMMARY"
  fi
fi

exit "$exitcode"
