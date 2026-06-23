# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ writeShellApplication, pkgs }:
let
  # Patched to forward each crate's `license` from cargo metadata into Cargo.nix
  # (upstream forwards `authors` but not `license`). buildCargoSbom reads it to
  # populate a concluded licence per crate for the SBOM (TR-03183-2 / CRA), with
  # no build and no import-from-derivation. doCheck is disabled because the patch
  # changes crate2nix's golden-output fixtures.
  crate2nix = pkgs.crate2nix.overrideAttrs (old: {
    patches = (old.patches or [ ]) ++ [ ./crate2nix-license.patch ];
    doCheck = false;
  });
in
writeShellApplication {
  name = "update-kata-cargo-nix";
  runtimeInputs = [
    crate2nix
  ]
  ++ (with pkgs; [
    nix
    cargo
    git
    coreutils
    gnused
    gawk
  ]);
  text = builtins.readFile ./update-kata-cargo-nix.sh;
}
