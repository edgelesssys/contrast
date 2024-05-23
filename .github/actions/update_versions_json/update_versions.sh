#!/bin/bash

# `create_hash` creates a hash for a given file.
function create_hash {
  nix hash file --sri --type sha256 "$1" || exit 1
}

# `update` appends a new version entry to a given section.
function update {
  echo $(
    cat ./versions.json | jq --arg NAME "$1" --arg HASH "$2" --arg VERSION "$VERSION" '.[$NAME] |= . + [{"version": $VERSION,hash: $HASH}]' || exit 1
  ) > ./versions.json
}

# `check_for_version` checks if the given entry already contains a version.
function check_for_version {
  out=$(cat ./versions.json | jq --arg NAME "$1" --arg VERSION "$VERSION" '.[$NAME] | map(select(.version == $VERSION))')
  if [[ ! "$out" = "[]" ]]; then
    echo "[x] Version $VERSION exists for entry $1"
    exit 1
  fi
}

echo "::group::Updating versions"

# check if the version environment variable is set
if [[ -z "$VERSION" ]]; then
  echo "[x] VERSION environment variable not set"
  exit 1
fi

# copy contrast from the symlink to be able to hash it
if [[ -L "./result-cli/bin/contrast" ]]; then
  cp $(readlink ./result-cli/bin/contrast) ./result-cli/bin/contrast-to-hash
else
  cp ./result-cli/bin/contrast ./result-cli/bin/contrast-to-hash
fi

# declare an associative array that pairs the field name
# in ./versions.json with the path to the file
declare -A fields
fields["contrast"]="./result-cli/bin/contrast-to-hash"
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

  echo "[*] Updating ./versions.json for $field"
  update "$field" "$hash"

  echo ""
done

echo "[*] Formatting ./versions.json"
cat ./versions.json | jq --indent 2 | tee versions.json > /dev/null

echo "::endgroup::"
