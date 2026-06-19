# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ lib }:

rec {
  # Do not recurse into these.
  plumbingAttrs = [
    "callPackage"
    "newScope"
    "overrideScope"
    "overrideScope'"
    "packages"
    "override"
    "overrideAttrs"
    "overrideDerivation"
    "__functor"
    "__functionArgs"
  ];

  # Returns a list of { name = "a/b/c"; path = drv; } pairs for every derivation reachable.
  # This is the shape linkFarm expects.
  collectDerivations =
    skip: attrs:
    let
      go =
        path: v:
        let
          eval = builtins.tryEval v;
          val = eval.value;
        in
        if lib.isDerivation val then
          [
            {
              name = lib.concatStringsSep "/" path;
              path = val;
            }
          ]
        else if lib.isAttrs val && !lib.isFunction val then
          lib.concatLists (
            lib.mapAttrsToList (
              n: c: if (lib.elem n plumbingAttrs) || (lib.elem n skip) then [ ] else go (path ++ [ n ]) c
            ) val
          )
        else
          [ ];
    in
    go [ ] attrs;

  buildableOn =
    hostPlatform: entries:
    lib.filter (
      e: lib.meta.availableOn hostPlatform e.path && e.path.system == hostPlatform.system
    ) entries;

  # Every reachable derivation that exposes a passthru.bombonVendoredSbom. These
  # are passed to bombon's buildBom as extraPaths: bombon discovers vendored
  # SBOMs by walking drvAttrs, but the matrix is a linkFarm that references its
  # contents as runtime paths, so the packages must be given as explicit roots.
  collectVendoredSbomPackages =
    skip: attrs:
    lib.concatMap (
      e:
      let
        has = builtins.tryEval (e.path ? bombonVendoredSbom);
      in
      lib.optional (has.success && has.value) e.path
    ) (collectDerivations skip attrs);

  mkMatrix =
    pkgs:
    pkgs.linkFarm "contrast-matrix" (
      buildableOn pkgs.stdenv.hostPlatform (
        collectDerivations [
          "contrastPkgsStatic"
          "contrast-releases"
          "matrix"
          "sbom"
        ] pkgs.contrastPkgs
      )
    );
}
