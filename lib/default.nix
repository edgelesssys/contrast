{ inputs }:

let
  mkLib =
    nixpkgs:
    nixpkgs.lib.extend (
      self: _: {
        ci = import ./ci.nix { lib = self; };
      }
    );
in
mkLib inputs.nixpkgs
