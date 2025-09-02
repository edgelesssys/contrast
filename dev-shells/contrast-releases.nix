# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  mkShellNoCC,
  contrastPkgs,
}:

let
  toDemoShell =
    version: contrast-release:
    lib.nameValuePair "demo-${version}" (mkShellNoCC {
      packages = [ contrast-release ];
      shellHook = ''
        cd "$(mktemp -d)"
        compgen -G "${contrast-release}/*.yml" > /dev/null && install -m644 ${contrast-release}/*.yml .
        [[ -d ${contrast-release}/deployment ]] && install -m644 -Dt ./deployment ${contrast-release}/deployment/*
        export DO_NOT_TRACK=1
      '';
    });
in

lib.mapAttrs' toDemoShell contrastPkgs.contrast-releases
