#! /usr/bin/env nix
#! nix shell .#base.nixpkgs.nix .#base.nixpkgs.gnused .#base.nixpkgs.bash --command bash
# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1
# shellcheck shell=bash

set -euo pipefail

scriptDir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)

for variant in go rust; do
  echo "Updating hash of kata.release-tarball.$variant" >&2
  oldHash="$(nix eval ".#base.kata.release-tarball.$variant.outputHash" --raw)"
  if [[ -n $oldHash ]]; then
    sed -i "s|$oldHash||g" "$scriptDir/package.nix"
  fi

  nixBuildFailure=$(nix build ".#base.kata.release-tarball.$variant" --no-link 2>&1 >/dev/null || true)
  newHash=$(echo "$nixBuildFailure" | grep got: | awk '{print $2}')

  sed -i "s|hash = \"\"|hash = \"$newHash\"|g" "$scriptDir/package.nix"
done
