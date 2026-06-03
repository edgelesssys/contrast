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
      url = "github:edgelesssys/bombon/8e8ebc8d1c38e5364bd5222c385c82a492de618a";
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
        contrastLib = import ./lib { inherit (nixpkgs) lib; };
        sets = contrastLib.mkSets { inherit nixpkgs self system; };

        pkgs = sets.base;

        treefmtEval = treefmt-nix.lib.evalModule pkgs ./treefmt.nix;
      in
      {
        devShells = pkgs.callPackages ./dev-shells {
          git-hooks-lib = git-hooks.lib.${system};
        };

        formatter = treefmtEval.config.build.wrapper;

        checks.formatting = treefmtEval.config.build.check self;

        legacyPackages = nixpkgs.lib.mapAttrs (_name: contrastLib.reverseContrastNesting) sets;
      }
    );

  nixConfig = {
    extra-substituters = [ "https://edgelesssys.cachix.org" ];
    extra-trusted-public-keys = [
      "edgelesssys.cachix.org-1:erQG/S1DxpvJ4zuEFvjWLx/4vujoKxAJke6lK2tWeB0="
    ];
  };
}
