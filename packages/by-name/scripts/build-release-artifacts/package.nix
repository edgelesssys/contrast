# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  writeShellApplication,
  coreutils,
  jq,
  contrastPkgs,
}:

# Usage: build-release-artifacts --image-replacements <file> --out <dir>
writeShellApplication {
  name = "build-release-artifacts";

  runtimeInputs = [
    coreutils
    jq
    contrastPkgs.contrast.resourcegen
    contrastPkgs.scripts.finalize-sbom
  ];
  text = builtins.readFile ./build-release-artifacts.sh;
}
