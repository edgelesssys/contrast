# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  inputs = {
    nixpkgs = {
      # Renovate has some false heuristic to detect whether something can be `nix flake update`ed,
      # see https://github.com/renovatebot/renovate/issues/29721 and
      # https://github.com/renovatebot/renovate/blob/743fed0ec6ca5810e274571c83fa6d4f5213d4e7/lib/modules/manager/nix/extract.ts#L6.
      # We must keep the following string in the file for renovate to work: "github:NixOS/nixpkgs/nixpkgs-unstable"
      url = "github:NixOS/nixpkgs?ref=nixos-unstable";
    };
    flake-utils = {
      url = "github:numtide/flake-utils";
    };
    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
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
          overlays = [ (import ./overlays/nixpkgs.nix) ];
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
