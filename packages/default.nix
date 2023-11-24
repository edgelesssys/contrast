{ pkgs }:
with pkgs;
{
  genpolicy = callPackage ./genpolicy.nix { };
}
