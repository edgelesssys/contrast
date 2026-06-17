# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ writeShellApplication, pkgs }:
writeShellApplication {
  name = "update-kata-cargo-nix";
  runtimeInputs = with pkgs; [
    nix
    cargo
    crate2nix
    git
    coreutils
    gnused
    gawk
  ];
  text = builtins.readFile ./update-kata-cargo-nix.sh;
}
