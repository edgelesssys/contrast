#!/usr/bin/env bash
# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

set -euo pipefail

readonly versionsFile="./packages/contrast-releases.json"

# check if the version environment variable is set
if [[ -z ${VERSION:-} ]]; then
  echo "[x] Error: VERSION environment variable not set" >&2
  exit 1
fi

# declare an associative array that pairs the field name
# in ./packages/versions.json with the path to the file
declare -A fields
fields["contrast"]="./result-cli/bin/contrast"
fields["coordinator.yml"]="./workspace/coordinator.yml"
fields["runtime.yml"]="./workspace/runtime.yml"
fields["emojivoto-demo.zip"]="./workspace/emojivoto-demo.zip"
fields["emojivoto-demo.yml"]="./workspace/emojivoto-demo.yml"
for platform in aks-clh-snp metal-qemu-tdx k3s-qemu-tdx metal-qemu-snp k3s-qemu-snp rke2-qemu-tdx; do
  fields["coordinator-${platform}.yml"]="./workspace/coordinator-${platform}.yml"
  fields["runtime-${platform}.yml"]="./workspace/runtime-${platform}.yml"
done

for field in "${!fields[@]}"; do
  # get the file path
  file=${fields["$field"]}

  # skip files which are not included in current release
  if [[ ! -f $file ]]; then
    continue
  fi

  out=$(
    jq --arg NAME "$field" \
      'if has($NAME) then . else . + {($NAME): []} end' \
      "${versionsFile}"
  )
  echo "$out" >"${versionsFile}"

  # check if any field contains the given version
  out=$(
    jq --arg NAME "$field" \
      --arg VERSION "$VERSION" \
      '.[$NAME] | map(select(.version == $VERSION))' \
      "${versionsFile}"
  )
  if [[ $out != "[]" ]]; then
    echo "[x] Error: version $VERSION exists for entry $field" >&2
    exit 1
  fi

  echo "[*] Creating hash for $file" >&2
  hash=$(nix hash file --sri --type sha256 "$(realpath "$file")")
  echo "      $hash" >&2

  echo "[*] Updating ${versionsFile} for $field" >&2
  out=$(
    jq --arg NAME "$field" \
      --arg HASH "$hash" \
      --arg VERSION "$VERSION" \
      '.[$NAME] |= . + [{"version": $VERSION,hash: $HASH}]' \
      "${versionsFile}"
  )
  echo "$out" >"${versionsFile}"

  echo ""
done

echo "[*] Formatting ${versionsFile}"
out=$(jq --indent 2 'to_entries | sort_by(.key) | from_entries' "${versionsFile}")
echo "$out" >"${versionsFile}"
