# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

# `create_hash` creates a hash for a given file.
function create_hash {
  nix hash file --sri --type sha256 "$1" || exit 1
}

# `update` appends a new version entry to a given section.
function update {
  out=$(jq --arg NAME "$1" --arg HASH "$2" --arg VERSION "$VERSION" '.[$NAME] |= . + [{"version": $VERSION,hash: $HASH}]' ./packages/versions.json || exit 1) 
  echo "$out" > ./packages/versions.json
}

# `check_for_version` checks if the given entry already contains a version.
function check_for_version {
  out=$(jq --arg NAME "$1" --arg VERSION "$VERSION" '.[$NAME] | map(select(.version == $VERSION))' ./packages/versions.json || exit 1)
  if [[ ! "$out" = "[]" ]]; then
    echo "[x] Version $VERSION exists for entry $1"
    exit 1
  fi
}

echo "::group::Updating versions"

# check if the version environment variable is set
if [[ ! -v VERSION ]]; then
  echo "[x] VERSION environment variable not bound"
  exit 1
fi
if [[ -z "$VERSION" ]]; then
  echo "[x] VERSION environment variable not set"
  exit 1
fi

# copy contrast from the symlink to be able to hash it
if [[ -L "./result-cli/bin/contrast" ]]; then
  cp "$(readlink ./result-cli/bin/contrast)" ./workspace/contrast-to-hash
else
  cp ./result-cli/bin/contrast ./workspace/contrast-to-hash
fi

# declare an associative array that pairs the field name
# in ./packages/versions.json with the path to the file
declare -A fields
fields["contrast"]="./workspace/contrast-to-hash"
fields["coordinator.yml"]="./workspace/coordinator.yml"
fields["runtime.yml"]="./workspace/runtime.yml"
fields["emojivoto-demo.zip"]="./workspace/emojivoto-demo.zip"


for field in "${!fields[@]}"
do
  # check if any field contains the given version
  check_for_version "$field" 

  # get the file path
  file=${fields["$field"]}

  echo "[*] Creating hash for $file"
  hash=$(create_hash "$file")
  echo "      $hash"

  echo "[*] Updating ./packages/versions.json for $field"
  update "$field" "$hash"

  echo ""
done

echo "[*] Formatting ./packages/versions.json"
out=$(jq --indent 2 . ./packages/versions.json) 
echo "$out" > ./packages/versions.json

echo "::endgroup::"
