# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  writeShellApplication,
  nix,
  jq,
  coreutils,
}:

# Usage: finalize-sbom <full|cli|runtimeConfidential|runtimeNonConfidential> [output-file]
writeShellApplication {
  name = "finalize-sbom";

  runtimeInputs = [
    nix
    jq
    coreutils
  ];
  text = builtins.readFile ./finalize-sbom.sh;
}
