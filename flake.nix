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
    crate2nix = {
      url = "github:edgelesssys/crate2nix/3597241ae1fb786945725ef72a6b439da717793a";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    bombon = {
      url = "github:edgelesssys/bombon/8f0412a597fcb64b418be74ca402e58b37e95dd2";
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

        # setsFromDirectory reads overlays from a directory and creates a set of pkgs instances for each.
        # The filename is used as attribute name in the resulting set, the .nix extension is stripped.
        setsFromDirectory =
          dir:
          builtins.listToAttrs (
            map (file: rec {
              name = builtins.substring 0 (builtins.stringLength file - 4) (baseNameOf file);
              value = mkSet ((defaultOverlays name) ++ [ (import (dir + "/${file}")) ]);
            }) (builtins.attrNames (builtins.readDir dir))
          );

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
          (final: _prev: {
            inherit (import self.inputs.bombon { pkgs = final; }) buildBom;
          })
          (final: _prev: {
            crate2nix = self.inputs.crate2nix.packages.${final.stdenv.hostPlatform.system}.default;
          })
          (_final: _prev: { runtimePkgs = self.legacyPackages.x86_64-linux.${set}; })
          (import ./overlays/nixpkgs.nix)
          (import ./overlays/contrast.nix)
          (import ./overlays/runtimepkgs.nix)
        ];

        sets = setsFromDirectory ./overlays/sets;

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
