#!/bin/bash

# `create_hash` creates a hash for a given file.
function create_hash {
  printf "  $1: "
  nix hash file --sri --type sha256 "$1" || exit 1
}

echo "::group::Updating versions"

# check if the version environment variable is set
if [[ -z "$VERSION" ]]; then
  echo "[x] VERSION environment variable not set"
  exit 1
fi

echo "[*] Updating versions.json"

echo "[*] Creating hashes"

contrast_hash=$(create_hash "./result-cli/bin/contrast")
printf "$contrast_hash\n"

coordinator_hash=$(create_hash "./workspace/coordinator.yml")
printf "$coordinator_hash\n"

runtime_hash=$(create_hash "./workspace/runtime.yml")
printf "$runtime_hash\n"

emojivoto_hash=$(create_hash "./workspace/emojivoto-demo.zip")
printf "$emojivoto_hash\n"

echo "::endgroup::"
