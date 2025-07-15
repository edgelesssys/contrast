# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ inputs }:
final: prev: {
  fenix = inputs.fenix.packages.${final.system};
  craneLib = (inputs.crane.mkLib prev.pkgs) // {
    prepareSource =
      {
        src,
        cargoDir,
        postPatch ? "",
      }:
      let
        patchedSrc = prev.pkgs.applyPatches {
          inherit postPatch;
          src =
            let
              additionalFilter = path: _type: builtins.match ".*[in|proto]$" path != null;
              cargoOrAdditional =
                path: type: (additionalFilter path type) || (final.craneLib.filterCargoSources path type);
            in
            prev.pkgs.lib.cleanSourceWith {
              inherit src;
              filter = cargoOrAdditional;
              name = "source";
            };
        };
      in
      {
        src = patchedSrc;
        cargoToml = "${patchedSrc}/${cargoDir}/Cargo.toml";
        cargoLock = "${patchedSrc}/${cargoDir}/Cargo.lock";
      };
  };
}
