# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ lib }:

rec {
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

  # orderedCombinations returns every non-empty ordered arrangement of distinct set names: all permutations of all non-empty subsets.
  # Order is significant, so "a+b" and "b+a" are distinct combinations that get applied in the order given.
  orderedCombinations = setNames: lib.concatMap permutations (nonEmptySubsets setNames);

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
  # For every ordered combination of the available sets it builds one set, named by joining the set names with "+", applying the set overlays in that order.
  mkSets =
    {
      nixpkgs,
      self,
      system,
    }:
    let
      setsDir = self + "/overlays/sets";
      setNames = map (lib.removeSuffix ".nix") (builtins.attrNames (builtins.readDir setsDir));
      # mkOrderedSet builds the package set for a single ordered combination of set names by layering the corresponding set overlays on top of the default overlays.
      mkOrderedSet =
        p:
        mkSet { inherit nixpkgs system; } (
          (defaultOverlays { inherit self; } (lib.concatStringsSep "+" p))
          ++ map (n: import (setsDir + "/${n}.nix")) p
        );
    in
    builtins.listToAttrs (
      map (p: {
        name = lib.concatStringsSep "+" p;
        value = mkOrderedSet p;
      }) (orderedCombinations setNames)
    );
}
