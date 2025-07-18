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
      # TODO(charludo): change to upstream once https://github.com/nix-community/fenix/pull/145 is merged
      url = "github:soywod/fenix?rev=c7af381484169a78fb79a11652321ae80b0f92a6";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
      treefmt-nix,
      ...
    }:
    flake-utils.lib.eachDefaultSystem (
      system:

      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [
            (final: _prev: { fenix = self.inputs.fenix.packages.${final.system}; })
            (import ./overlays/nixpkgs.nix)
          ];
          config.allowUnfree = true;
          config.nvidia.acceptLicense = true;
        };
        inherit (pkgs) lib;
        treefmtEval = treefmt-nix.lib.evalModule (pkgs // ourPkgs) ./treefmt.nix;
        ourPkgs = import ./packages { inherit pkgs lib; };
      in

      {
        devShells = pkgs.callPackages ./dev-shells { inherit (ourPkgs) contrast-releases; };

        formatter = treefmtEval.config.build.wrapper;

        checks.formatting = treefmtEval.config.build.check self;

        legacyPackages = pkgs // ourPkgs;
      }
    );

  nixConfig = {
    extra-substituters = [ "https://edgelesssys.cachix.org" ];
    extra-trusted-public-keys = [
      "edgelesssys.cachix.org-1:erQG/S1DxpvJ4zuEFvjWLx/4vujoKxAJke6lK2tWeB0="
    ];
  };
}
