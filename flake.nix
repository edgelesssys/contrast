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
        };
        inherit (pkgs) lib;
        treefmtEval = treefmt-nix.lib.evalModule pkgs ./treefmt.nix;
        ourPkgs = import ./packages { inherit pkgs lib; };
      in
      {
        devShells =
          {
            default = pkgs.mkShell {
              packages = with pkgs; [
                azure-cli
                crane
                delve
                go
                golangci-lint
                gopls
                gotools
                just
                kubectl
              ];
              shellHook = ''
                alias make=just
                export DO_NOT_TRACK=1
              '';
            };
            docs = pkgs.mkShell {
              packages = with pkgs; [ yarn ];
              shellHook = ''
                yarn install
              '';
            };
          }
          // (
            let
              toDemoShell =
                version: contrast-release:
                lib.nameValuePair "demo-${version}" (
                  pkgs.mkShell {
                    packages = [ contrast-release ];
                    shellHook = ''
                      cd "$(mktemp -d)"
                      [[ -e ${contrast-release}/runtime.yml ]] && install -m644 ${contrast-release}/runtime.yml .
                      [[ -e ${contrast-release}/coordinator.yml ]] && install -m644 ${contrast-release}/coordinator.yml .
                      [[ -d ${contrast-release}/deployment ]] && install -m644 -Dt ./deployment ${contrast-release}/deployment/*
                      export DO_NOT_TRACK=1
                    '';
                  }
                );
            in
            lib.mapAttrs' toDemoShell ourPkgs.contrast-releases
          );

        formatter = treefmtEval.config.build.wrapper;

        checks = {
          formatting = treefmtEval.config.build.check self;
        };

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
