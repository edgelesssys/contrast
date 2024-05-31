#!/usr/bin/env bash
# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

set -euo pipefail

# check if the version environment variable is set
if [[ -z "${VERSION:-}" ]]; then
  echo "[x] VERSION environment variable not set"
  exit 1
fi

# declare an associative array that pairs the field name
# in ./packages/versions.json with the path to the file
declare -A fields
fields["contrast"]="./result-cli/bin/contrast"
fields["coordinator.yml"]="./workspace/coordinator.yml"
fields["runtime.yml"]="./workspace/runtime.yml"
fields["emojivoto-demo.zip"]="./workspace/emojivoto-demo.zip"

for field in "${!fields[@]}"; do
  # check if any field contains the given version
  out=$(
    jq --arg NAME "$field" \
      --arg VERSION "$VERSION" \
      '.[$NAME] | map(select(.version == $VERSION))' \
      ./packages/versions.json
  )
  if [[ ! "$out" = "[]" ]]; then
    echo "[x] Version $VERSION exists for entry $field"
    exit 1
  fi

  # get the file path
  file=${fields["$field"]}

  echo "[*] Creating hash for $file"
  hash=$(nix hash file --sri --type sha256 "$(realpath "$file")")
  echo "      $hash"

  echo "[*] Updating ./packages/versions.json for $field"
  out=$(
    jq --arg NAME "$field" \
      --arg HASH "$hash" \
      --arg VERSION "$VERSION" \
      '.[$NAME] |= . + [{"version": $VERSION,hash: $HASH}]' \
      ./packages/versions.json
  )
  echo "$out" >./packages/versions.json

  echo ""
done

echo "[*] Formatting ./packages/versions.json"
out=$(jq --indent 2 . ./packages/versions.json)
echo "$out" >./packages/versions.json
