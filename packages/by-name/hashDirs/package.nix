# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  stdenvNoCC,
  nix,
}:

{ name, dirs }:

stdenvNoCC.mkDerivation {
  inherit name;
  dontUnpack = true;
  nativeBuildInputs = [ nix ];
  buildPhase = ''
    nix --extra-experimental-features nix-command hash path ${lib.concatStringsSep " " dirs} |
      LC_ALL=C sort |
      sha256sum |
      cut -d' ' -f1 > $out
  '';
}
