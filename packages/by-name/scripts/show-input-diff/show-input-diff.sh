#!/usr/bin/env bash
# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

maxDepth=999
set_name=base
system=$(nix eval --impure --raw --expr builtins.currentSystem)
new_args=()
while [[ $# -gt 0 ]]; do
  case $1 in
  --max-depth)
    maxDepth="$2"
    shift 2
    ;;
  --max-depth=*)
    maxDepth="${1#*=}"
    shift
    ;;
  --set)
    set_name="$2"
    shift 2
    ;;
  --set=*)
    set_name="${1#*=}"
    shift
    ;;
  --system)
    system="$2"
    shift 2
    ;;
  --system=*)
    system="${1#*=}"
    shift
    ;;
  *)
    new_args+=("$1")
    shift
    ;;
  esac
done
set -- "${new_args[@]}"

attr="legacyPackages.$system.$set_name.matrix"
left=$(nix eval --raw "${1:-github:edgelesssys/contrast}#$attr.drvPath" | tr -d '\n')
right=$(nix eval --raw "${2:-.}#$attr.drvPath" | tr -d '\n')
nix-diff "$left" "$right" --json | jq -r --argjson maxDepth "$maxDepth" '
  def printTree(level):
    (
      select(.drvName != null and .drvName != "") | ("  " * level) + .drvName,
      (.drvNames // [] | .[] | ("  " * (level + 1)) + .),
      (if level + 1 < $maxDepth then (.drvDiff.inputsDiff.inputDerivationDiffs // [] | .[] | printTree(level + 1)) else empty end)
    );
  .inputsDiff.inputDerivationDiffs[] | printTree(0)
'
