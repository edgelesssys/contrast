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

  # Returns the `passthru.sbom` of every reachable derivation that exposes one,
  # as a list of { name = "a/b/c"; sbom = drv; } pairs. Packages opt into the
  # aggregate SBOM simply by setting passthru.sbom; the top-level sbom package
  # collects them without needing to know about each one.
  collectSboms =
    skip: attrs:
    lib.concatMap (
      e:
      let
        has = builtins.tryEval (e.path ? sbom && e.path.sbom != null);
      in
      lib.optional (has.success && has.value) {
        inherit (e) name;
        inherit (e.path) sbom;
      }
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
