# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  writeShellApplication,
  nix-diff,
  jq,
}:

# show-input-diff shows which input derivations differ between two builds of the matrix output.
# LEFT and RIGHT pick flake refs. Defaults to upstream vs. local checkout.
writeShellApplication {
  name = "show-input-diff";
  text = builtins.readFile ./show-input-diff.sh;
  runtimeInputs = [
    nix-diff
    jq
  ];
}
