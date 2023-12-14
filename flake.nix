{
  inputs = {
    nixpkgs = {
      url = "nixpkgs/nixos-unstable";
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
    }: flake-utils.lib.eachDefaultSystem (system:
    let
      pkgs = import nixpkgs { inherit system; };
      inherit (pkgs) lib;

      goVendorHash = "sha256-jwN90izTTCqDTkWhMcm0YlDUN8+2FSEprK0JeX/7fp4=";

      treefmtEval = treefmt-nix.lib.evalModule pkgs ./treefmt.nix;
    in
    {
      packages = import ./packages { inherit pkgs goVendorHash; };

      devShells.default = pkgs.mkShell {
        packages = with pkgs; [ just ];
        shellHook = ''alias make=just'';
      };

      formatter = treefmtEval.config.build.wrapper;

      legacyPackages = pkgs;
    });
}
