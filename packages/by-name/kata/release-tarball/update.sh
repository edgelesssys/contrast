#! /usr/bin/env nix
#! nix shell .#nixpkgs.nix .#nixpkgs.gnused .#nixpkgs.bash --command bash
# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1
# shellcheck shell=bash

set -euo pipefail

scriptDir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)

oldHash="$(nix eval .#kata.release-tarball.outputHash --raw)"
sed -i "s|$oldHash||g" "$scriptDir/package.nix"

nixBuildFailure=$(nix build .#kata.release-tarball --no-link 2>&1 >/dev/null || true)
newHash=$(echo "$nixBuildFailure" | grep got: | awk '{print $2}')

sed -i "s|hash = \"\"|hash = \"$newHash\"|g" "$scriptDir/package.nix"
