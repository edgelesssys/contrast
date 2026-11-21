# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  writeShellApplication,
  coreutils,
  jq,
  nix,
  contrastPkgs,
}:

# Usage: build-release-cli --system <system> --out <dir>
writeShellApplication {
  name = "build-release-cli";

  runtimeInputs = [
    coreutils
    jq
    nix
    contrastPkgs.scripts.finalize-sbom
  ];
  text = builtins.readFile ./build-release-cli.sh;
}
