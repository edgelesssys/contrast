#!/usr/bin/env bash
# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

repo=$(git rev-parse --show-toplevel)
out="$repo/packages/by-name/kata/source/Cargo.nix"

echo "Building patched kata source" >&2
src=$(cd "$repo" && nix build --no-link --print-out-paths .#base.kata.source.src)

workdir=$(mktemp -d)
trap 'rm -rf "$workdir"' EXIT
cp -r --no-preserve=mode,ownership "$src"/. "$workdir/"
cd "$workdir"

echo "Running crate2nix generate" >&2
crate2nix generate -f Cargo.toml -o Cargo.nix

# crate2nix hardcodes crate sources as relative paths, rewrite them to resolve against workspaceSrc
sed -i \
  -e 's|^{ nixpkgs ? <nixpkgs>$|{ workspaceSrc\n, nixpkgs ? <nixpkgs>|' \
  -e 's|src = \./|src = workspaceSrc + "/|' \
  -e 's|src = workspaceSrc + "/\([^;]*\);|src = workspaceSrc + "/\1";|' \
  Cargo.nix

# crate2nix includes the absolute generation directory into safe-path's package id.
# Normalize it to a stable placeholder so regenerations don't produce diff.
sed -i "s|$workdir|/tmp/c2n-gen|g" Cargo.nix

# safe-path is both a workspace member and a crates.io crate of the same version,
# so crate2nix emits two definitions of it. Drop the registry one and point its references at the workspace member.
awk '
  /^      "registry\+https:\/\/github.com\/rust-lang\/crates.io-index#safe-path@0.1.0" = rec \{$/ { skip = 1 }
  skip { if ($0 == "      };") skip = 0; next }
  { print }
' Cargo.nix >Cargo.nix.tmp && mv Cargo.nix.tmp Cargo.nix
sed -i \
  's|registry+https://github.com/rust-lang/crates.io-index#safe-path@0.1.0|path+file:///tmp/c2n-gen/src/libs/safe-path#0.1.0|g' \
  Cargo.nix

# crate2nix' test-runner derivation writes its log directly to $out via tee, so the default installPhase which mkdirs $out collides
sed -i '/buildInputs = testInputs;/a\            dontInstall = true;' Cargo.nix

cp Cargo.nix "$out"
echo "Wrote $out" >&2
