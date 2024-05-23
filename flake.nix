# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  inputs = {
    nixpkgs = {
      url = "github:NixOS/nixpkgs/nixpkgs-unstable";
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
    { self
    , nixpkgs
    , flake-utils
    , treefmt-nix
    , ...
    }: flake-utils.lib.eachDefaultSystem (system:
    let
      pkgs = import nixpkgs {
        inherit system;
        overlays = [ (import ./overlays/nixpkgs.nix) ];
      };
      inherit (pkgs) lib;
      treefmtEval = treefmt-nix.lib.evalModule pkgs ./treefmt.nix;
    in
    {
      devShells = {
        default = pkgs.mkShell {
          packages = with pkgs; [
            delve
            go
            golangci-lint
            gopls
            gotools
            just
          ];
          shellHook = ''
            alias make=just
            export DO_NOT_TRACK=1
          '';
        };
        docs = pkgs.mkShell {
          packages = with pkgs; [
            yarn
          ];
          shellHook = ''
            yarn install
          '';
        };
        demo = pkgs.mkShell rec {
          packages = [];

          json = builtins.fromJSON (builtins.readFile ./versions.json);

          # fetch the required contrast sources
          version = builtins.readFile ./version.txt;

          # select the correct hashes
          contrastHash = builtins.elemAt (builtins.filter (a: a.version == version) json.contrast) 0

          contrast = pkgs.fetchurl {
            url = "https://github.com/edgelesssys/contrast/releases/download/${version}/contrast";
            hash = "sha256-bxUIis/6uKTdqOa/uILLGOs0M2XqMkrq371EfnwsvtQ=";
          };
          coordinator = pkgs.fetchurl {
            url = "https://github.com/edgelesssys/contrast/releases/download/${version}/coordinator.yml";
            hash = "sha256-W4K5UJYwBXGxLZ4EJVymHW+Zoc57rDLHfCbQIFic03E=";
          };
          emojivoto = pkgs.fetchzip {
            url = "https://github.com/edgelesssys/contrast/releases/download/${version}/emojivoto-demo.zip";
            hash = "sha256-MGmN/6lPvGvbrXXvI1z8eUx2qsE8f5NewjTP4Jk5l6U=";
          };

          shellHook = ''
            mkdir -p demo
            cp ${contrast} ./demo/contrast
            cp ${coordinator} ./demo/coordinator.yml
            cp -r ${emojivoto} ./demo/deployment
          '';
        };
      };

      formatter = treefmtEval.config.build.wrapper;

      checks = {
        formatting = treefmtEval.config.build.check self;
      };

      legacyPackages = pkgs // (import ./packages { inherit pkgs lib; });
    });

  nixConfig = {
    extra-substituters = [
      "https://edgelesssys.cachix.org"
    ];
    extra-trusted-public-keys = [
      "edgelesssys.cachix.org-1:erQG/S1DxpvJ4zuEFvjWLx/4vujoKxAJke6lK2tWeB0="
    ];
  };
}
