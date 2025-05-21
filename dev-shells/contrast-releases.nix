# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  mkShellNoCC,
  contrast-releases,
}:

let
  toDemoShell =
    version: contrast-release:
    lib.nameValuePair "demo-${version}" (mkShellNoCC {
      packages = [ contrast-release ];
      shellHook = ''
        cd "$(mktemp -d)"
        [[ -e ${contrast-release}/runtime.yml ]] && install -m644 ${contrast-release}/runtime.yml .
        compgen -G "${contrast-release}/runtime-*.yml" > /dev/null && install -m644 ${contrast-release}/runtime-*.yml .
        [[ -e ${contrast-release}/coordinator.yml ]] && install -m644 ${contrast-release}/coordinator.yml .
        compgen -G "${contrast-release}/coordinator-*.yml" > /dev/null && install -m644 ${contrast-release}/coordinator-*.yml .
        [[ -d ${contrast-release}/deployment ]] && install -m644 -Dt ./deployment ${contrast-release}/deployment/*
        export DO_NOT_TRACK=1
      '';
    });
in

lib.mapAttrs' toDemoShell contrast-releases
