# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ lib }:

rec {
  # maxCombineDepth bounds how many sets may be combined at once via the `+` syntax.
  maxCombineDepth = 3;

  # nonEmptySubsets returns every non-empty subset of the given list.
  nonEmptySubsets =
    list:
    builtins.filter (s: s != [ ]) (lib.foldl' (acc: x: acc ++ map (s: s ++ [ x ]) acc) [ [ ] ] list);

  # permutations returns all orderings of the given list.
  permutations =
    list:
    if list == [ ] then
      [ [ ] ]
    else
      lib.concatLists (
        lib.imap0 (i: x: map (p: [ x ] ++ p) (permutations (lib.take i list ++ lib.drop (i + 1) list))) list
      );

  # canonicalName joins a subset of set names into a single, stable name by sorting the names alphabetically and separating them with "+".
  # Every permutation of the same set names therefore maps to the same canonical name.
  canonicalName = subset: lib.concatStringsSep "+" (lib.sort builtins.lessThan subset);

  # combinableSubsets returns all setNames that may be used as the flake entry point, i.e. non-empty and no deeper than maxCombineDepth.
  combinableSubsets =
    setNames: builtins.filter (s: builtins.length s <= maxCombineDepth) (nonEmptySubsets setNames);

  # mkSet instantiates nixpkgs for the given system with the given overlays.
  mkSet =
    { nixpkgs, system }:
    overlays:
    import nixpkgs {
      inherit system overlays;
      config.allowUnfree = true;
      config.nvidia.acceptLicense = true;
    };

  # reverseContrastNesting takes a pkgs instance and reverses the nesting by moving the
  # contrastPkgs attributes to the top level and the originally top-level nixpkgs attributes
  # to a nested nixpkgs attribute. This allows easy access to contrastPkgs via the nix flake
  # CLI while still exposing the overlayed packages from nixpkgs under the nixpkgs attribute.
  reverseContrastNesting =
    pkgs:
    pkgs.contrastPkgs
    // {
      nixpkgs = removeAttrs pkgs [
        "fenix"
        "contrastPkgs"
      ];
    };

  # defaultOverlays is the overlay stack shared by every set.
  defaultOverlays =
    { self, ... }:
    set: [
      (final: _prev: { fenix = self.inputs.fenix.packages.${final.stdenv.hostPlatform.system}; })
      (final: _prev: {
        inherit (import self.inputs.bombon { pkgs = final; }) buildBom;
      })
      (final: _prev: {
        crate2nix = self.inputs.crate2nix.packages.${final.stdenv.hostPlatform.system}.default;
      })
      (_final: _prev: { runtimePkgs = self.legacyPackages.x86_64-linux.${set}; })
      (import (self + "/overlays/nixpkgs.nix"))
      (import (self + "/overlays/contrast.nix"))
      (import (self + "/overlays/runtimepkgs.nix"))
    ];

  # mkSets builds the full attrset of package sets discovered under overlays/sets.
  # For each combinable subset it builds the canonicalName set and then exposes it under every permutation of the subset's names joined with "+".
  mkSets =
    {
      nixpkgs,
      self,
      system,
    }:
    let
      setsDir = self + "/overlays/sets";
      setNames = map (lib.removeSuffix ".nix") (builtins.attrNames (builtins.readDir setsDir));
      # mkCanonicalSet builds the package set for a single subset of set names by layering the corresponding set overlays on top of the default overlays.
      mkCanonicalSet =
        s:
        mkSet { inherit nixpkgs system; } (
          (defaultOverlays { inherit self; } (canonicalName s)) ++ map (n: import (setsDir + "/${n}.nix")) s
        );
      subsets = combinableSubsets setNames;
      canonicalSets = builtins.listToAttrs (
        map (s: {
          name = canonicalName s;
          value = mkCanonicalSet s;
        }) subsets
      );
    in
    builtins.listToAttrs (
      lib.concatMap (
        s:
        map (p: {
          name = lib.concatStringsSep "+" p;
          value = canonicalSets.${canonicalName s};
        }) (permutations s)
      ) subsets
    );
}
