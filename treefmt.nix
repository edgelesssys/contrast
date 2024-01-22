_:
{
  projectRootFile = "flake.nix";
  programs = {
    # keep-sorted start
    deadnix.enable = true;
    just.enable = true;
    keep-sorted.enable = true;
    nixpkgs-fmt.enable = true;
    shellcheck.enable = true;
    shfmt.enable = true;
    statix.enable = true;
    yamlfmt.enable = true;
    # keep-sorted end
  };
}
