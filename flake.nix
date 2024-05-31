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

          json = builtins.fromJSON (builtins.readFile ./packages/versions.json);

          version = (lib.lists.last json.contrast).version; # select the latest contrast version

          # select all hashes based on the extracted version; since no "error" version exists the download will fail
          # if the given version doesn't exist for a file.
          contrastHash = (lib.lists.findFirst (obj: obj.version == version) "error" json.contrast).hash;
          coordinatorHash = (lib.lists.findFirst (obj: obj.version == version) "error" json."coordinator.yml").hash;
          runtimeHash = (lib.lists.findFirst (obj: obj.version == version) "error" json."runtime.yml").hash;
          emojivotoHash = (lib.lists.findFirst (obj: obj.version == version) "error" json."emojivoto-demo.zip").hash;

          contrast = pkgs.fetchurl {
            url = "https://github.com/edgelesssys/contrast/releases/download/${version}/contrast";
            hash = builtins.trace "contrast hash: ${contrastHash}" contrastHash;
          };
          coordinator = pkgs.fetchurl {
            url = "https://github.com/edgelesssys/contrast/releases/download/${version}/coordinator.yml";
            hash = builtins.trace "coordinator hash: ${coordinatorHash}" coordinatorHash;
          };
          runtime = pkgs.fetchurl {
            url = "https://github.com/edgelesssys/contrast/releases/download/${version}/runtime.yml";
            hash = builtins.trace "runtime hash: ${runtimeHash}" runtimeHash;
          };
          emojivoto = pkgs.fetchzip {
            url = "https://github.com/edgelesssys/contrast/releases/download/${version}/emojivoto-demo.zip";
            hash = builtins.trace "emojivoto hash: ${emojivotoHash}" emojivotoHash;
          };

          shellHook = ''
            cd "$(mktemp -d)" # create a temporary demodir

            # copy everything over
            cp ${contrast} ./contrast
            cp ${coordinator} ./coordinator.yml
            cp ${runtime} ./runtime.yml

            mkdir -p deployment
            cp -r ${emojivoto} ./deployment
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
