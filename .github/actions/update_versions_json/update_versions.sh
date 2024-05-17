#!/bin/bash

# `create_hash` creates a hash for a given file.
function create_hash {
  nix hash file --sri --type sha256 "$1" || exit 1
}

# `append_to` appends a new version entry to a given section.
function append_to {
  echo $(
    cat versions.json | jq --arg NAME "$1" --arg HASH "$2" --arg VERSION "$VERSION" '.[$NAME] |= . + [{"version": $VERSION,hash: $HASH}]' || exit 1
  ) > versions.json
}

echo "::group::Updating versions"

# check if the version environment variable is set
if [[ -z "$VERSION" ]]; then
  echo "[x] VERSION environment variable not set"
  exit 1
fi

echo "[*] Creating hashes"
contrast_hash=$(create_hash "./result-cli/bin/contrast")
echo "  contrast: $contrast_hash"
coordinator_hash=$(create_hash "./workspace/coordinator.yml")
echo "  coordinator.yml: $coordinator_hash"
runtime_hash=$(create_hash "./workspace/runtime.yml")
echo "  runtime.yml: $runtime_hash"
emojivoto_hash=$(create_hash "./workspace/emojivoto-demo.zip")
echo "  emojivoto-demo.zip: $emojivoto_hash"

echo "[*] Updating versions.json"
append_to "contrast" "$contrast_hash"
append_to "coordinator.yml" "$coordinator_hash"
append_to "runtime.yml" "$runtime_hash"
append_to "emojivoto-demo.zip" "$emojivoto_hash"

echo "[*] Formatting versions.json"
cat versions.json | jq --indent 2 | tee versions.json > /dev/null

echo "::endgroup::"
