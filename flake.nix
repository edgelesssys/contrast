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
        demo =
          let
            custom-packages = import ./packages { inherit pkgs lib; };
            json = builtins.fromJSON (builtins.readFile ./packages/contrast-releases.json);
            demoShell = {version, hash}: {
              name = builtins.replaceStrings ["."] ["-"] version;
              value =
              pkgs.mkShell {
                packages = [ custom-packages.contrast-releases.${builtins.replaceStrings ["."] ["-"] version} ];
              };
            };
          in
          builtins.listToAttrs (builtins.map demoShell json.contrast);
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
