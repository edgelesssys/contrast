# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  writeShellApplication,
  git,
}:

writeShellApplication {
  name = "nixos-image-diff";
  text = builtins.readFile ./nixos-image-diff.sh;
  runtimeInputs = [
    git
  ];
}
