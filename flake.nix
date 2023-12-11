{
  inputs = {
    nixpkgs.url = "nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    { self
    , flake-utils
    , nixpkgs
    }: flake-utils.lib.eachDefaultSystem (system:
    let
      pkgs = import nixpkgs { inherit system; };
      inherit (pkgs) lib;

      goVendorHash = "sha256-jwN90izTTCqDTkWhMcm0YlDUN8+2FSEprK0JeX/7fp4=";
    in
    {
      packages = import ./packages { inherit pkgs goVendorHash; };

      devShells = {
        default = pkgs.mkShell {
          packages = with pkgs; [ just ];
          shellHook = ''alias make=just'';
        };
      };
    });
}
