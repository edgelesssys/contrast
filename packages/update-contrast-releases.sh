#!/usr/bin/env bash
# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

readonly versionsFile="./packages/contrast-releases.json"

# check if the version environment variable is set
if [[ -z ${VERSION:-} ]]; then
  echo "[x] Error: VERSION environment variable not set" >&2
  exit 1
fi

platforms=(
  "metal-qemu-snp-gpu"
  "metal-qemu-snp"
  "metal-qemu-tdx"
  "metal-qemu-tdx-gpu"
)

target_configs=(
  "k3s"
)

# declare an associative array that pairs the field name
# in ./packages/versions.json with the path to the file
declare -A fields
fields["contrast-x86_64-linux"]="./workspace/artifacts/contrast-x86_64-linux"
fields["contrast-aarch64-darwin"]="./workspace/artifacts/contrast-aarch64-darwin"
fields["coordinator.yml"]="./workspace/artifacts/coordinator.yml"
fields["emojivoto-demo.yml"]="./workspace/artifacts/emojivoto-demo.yml"
fields["mysql-demo.yml"]="./workspace/artifacts/mysql-demo.yml"
fields["vault-demo.yml"]="./workspace/artifacts/vault-demo.yml"
for platform in "${platforms[@]}"; do
  fields["coordinator-${platform}.yml"]="./workspace/artifacts/coordinator-${platform}.yml"
  fields["runtime-${platform}.yml"]="./workspace/artifacts/runtime-${platform}.yml"
done
for target in "${target_configs[@]}"; do
  fields["node-installer-target-config-${target}.yml"]="./workspace/artifacts/node-installer-target-config-${target}.yml"
done

for field in "${!fields[@]}"; do
  # get the file path
  file=${fields["$field"]}

  # skip files which are not included in current release
  if [[ ! -f $file ]]; then
    echo "[*] Skipping $file as it does not exist" >&2
    echo >&2
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

  echo >&2
done

echo "[*] Formatting ${versionsFile}"
out=$(jq --indent 2 'to_entries | sort_by(.key) | from_entries' "${versionsFile}")
echo "$out" >"${versionsFile}"
