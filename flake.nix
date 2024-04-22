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
