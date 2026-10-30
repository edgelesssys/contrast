#!/usr/bin/env bash
# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

usage="usage: vuln-ignore-fill <vuln-ignores.toml> <findings.json>..."
ignores="${1:?$usage}"
shift
[[ $# -gt 0 ]] || {
  echo "$usage" >&2
  exit 1
}

updated=$(mktemp)
trap 'rm -f "$updated"' EXIT

# The findings come from scans that already applied the ignore list, so every finding is new.
added=0
while IFS= read -r entry; do
  table=$(jq -r .table <<<"$entry")
  package=$(jq -r '.package // ""' <<<"$entry")
  block=$(jq -r .block <<<"$entry")
  pos=$(awk -v table="$table" -v package="$package" -f "$POSITION_AWK" "$ignores")
  if [[ $pos -eq 0 ]]; then
    printf '%s\n' "$block" >>"$ignores"
  else
    {
      head -n $((pos - 1)) "$ignores"
      printf '%s\n' "$block"
      tail -n +"$pos" "$ignores"
    } >"$updated"
    cat "$updated" >"$ignores"
  fi
  added=$((added + 1))
done < <(jq -cs -f "$ENTRIES_JQ" "$@")

if [[ $added -eq 0 ]]; then
  echo "vuln-ignore-fill: no findings to add"
else
  echo "vuln-ignore-fill: added ${added} entries to ${ignores}, fill in their reasons"
fi
