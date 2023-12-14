{ pkgs, ... }:
{
  projectRootFile = "flake.nix";
  programs = {
    nixpkgs-fmt.enable = true;
    deadnix.enable = true;
  };
}
