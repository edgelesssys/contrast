# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  inputs = {
    nixpkgs = {
      url = "github:NixOS/nixpkgs?ref=nixos-unstable";
    };
    flake-utils = {
      url = "github:numtide/flake-utils";
    };
    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    fenix = {
      url = "github:nix-community/fenix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    git-hooks = {
      url = "github:cachix/git-hooks.nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
      treefmt-nix,
      git-hooks,
      ...
    }:
    flake-utils.lib.eachDefaultSystem (
      system:

      let
        # mkSet creates a set of packages based on a given set of overlays.
        mkSet =
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

        defaultOverlays = set: [
          (final: _prev: { fenix = self.inputs.fenix.packages.${final.stdenv.hostPlatform.system}; })
          (_final: _prev: { runtimePkgs = self.legacyPackages.x86_64-linux.${set}; })
          (import ./overlays/nixpkgs.nix)
          (import ./overlays/contrast.nix)
          (import ./overlays/runtimepkgs.nix)
        ];

        setsDir = ./overlays/sets;
        setNames = map (nixpkgs.lib.removeSuffix ".nix") (builtins.attrNames (builtins.readDir setsDir));

        nonEmptySubsets =
          list:
          builtins.filter (s: s != [ ]) (
            nixpkgs.lib.foldl' (acc: x: acc ++ map (s: [ x ] ++ s) acc) [ [ ] ] list
          );

        permutations =
          list:
          if list == [ ] then
            [ [ ] ]
          else
            nixpkgs.lib.concatLists (
              nixpkgs.lib.imap0 (
                i: x: map (p: [ x ] ++ p) (permutations (nixpkgs.lib.take i list ++ nixpkgs.lib.drop (i + 1) list))
              ) list
            );

        canonicalName =
          subset: nixpkgs.lib.concatStringsSep "+" (nixpkgs.lib.sort builtins.lessThan subset);

        subsets = nonEmptySubsets setNames;

        canonicalSets = builtins.listToAttrs (
          map (s: {
            name = canonicalName s;
            value = mkSet ((defaultOverlays (canonicalName s)) ++ map (n: import (setsDir + "/${n}.nix")) s);
          }) subsets
        );

        sets = builtins.listToAttrs (
          nixpkgs.lib.concatMap (
            s:
            map (p: {
              name = nixpkgs.lib.concatStringsSep "+" p;
              value = canonicalSets.${canonicalName s};
            }) (permutations s)
          ) subsets
        );

        pkgs = sets.base;

        treefmtEval = treefmt-nix.lib.evalModule pkgs ./treefmt.nix;
      in

      {
        devShells = pkgs.callPackages ./dev-shells {
          git-hooks-lib = git-hooks.lib.${system};
        };

        formatter = treefmtEval.config.build.wrapper;

        checks.formatting = treefmtEval.config.build.check self;

        legacyPackages = nixpkgs.lib.mapAttrs (_name: reverseContrastNesting) sets;
      }
    );

  nixConfig = {
    extra-substituters = [ "https://edgelesssys.cachix.org" ];
    extra-trusted-public-keys = [
      "edgelesssys.cachix.org-1:erQG/S1DxpvJ4zuEFvjWLx/4vujoKxAJke6lK2tWeB0="
    ];
  };
}
