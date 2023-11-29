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
    in
    {
      packages = {
        generate = pkgs.writeShellApplication {
          name = "generate";
          runtimeInputs = with pkgs; [ go protobuf protoc-gen-go protoc-gen-go-grpc ];
          text = ''
            go generate ./...
          '';
        };
      } // import ./packages { inherit pkgs; };
    });
}
