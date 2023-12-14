{ ... }:
{
  projectRootFile = "flake.nix";
  programs = {
    nixpkgs-fmt.enable = true;
    deadnix.enable = true;
    shellcheck.enable = true;
    shfmt.enable = true;
  };
}
