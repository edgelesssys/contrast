#!/usr/bin/env bash
# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

repo=$(git rev-parse --show-toplevel)
out="$repo/packages/by-name/kata/source/Cargo.nix"
hashes="$repo/packages/by-name/kata/source/crate-hashes.json"

echo "Building patched kata source" >&2
src=$(nix build --no-link --print-out-paths "$repo#base.kata.source.src")

workdir=$(mktemp -d)
trap 'rm -rf "$workdir"' EXIT
cp -r --no-preserve=mode,ownership "$src"/. "$workdir/"
cd "$workdir"

if [[ -f $hashes ]]; then
  cp "$hashes" "$workdir/crate-hashes.json"
fi

echo "Running crate2nix generate" >&2
crate2nix generate -f Cargo.toml -o Cargo.nix

# crate2nix emits each crate's `src` as a path relative to the *generation* directory, which is a nix store path, not the local path.
# At build time, it can not be referenced via a nix relative path.
sed -i \
  -e 's|^{ nixpkgs ? <nixpkgs>$|{ workspaceSrc\n, nixpkgs ? <nixpkgs>|' \
  -e 's|src = \./|src = workspaceSrc + "/|' \
  -e 's|src = workspaceSrc + "/\([^;]*\);|src = workspaceSrc + "/\1";|' \
  Cargo.nix

# crate2nix' test-runner derivation writes its log directly to $out via tee, so the default installPhase which mkdirs $out collides.
# a\ appends a line after the match.
sed -i '/buildInputs = testInputs;/a\            dontInstall = true;' Cargo.nix

cp Cargo.nix "$out"
echo "Wrote $out" >&2

cp crate-hashes.json "$hashes"
echo "Wrote $hashes" >&2
