# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ lib, pkgs, ... }:
{
  projectRootFile = "flake.nix";
  programs = {
    # keep-sorted start block=true
    actionlint.enable = true;
    deadnix.enable = true;
    formatjson5 = {
      enable = true;
      indent = 2;
      oneElementLines = true;
      sortArrays = true;
    };
    just.enable = true;
    keep-sorted.enable = true;
    nixfmt.enable = true;
    shellcheck.enable = true;
    shfmt.enable = true;
    statix.enable = true;
    terraform.enable = true;
    yamlfmt = {
      enable = true;
      settings.formatter.retain_line_breaks_single = true;
    };
    # keep-sorted end
  };
  settings.formatter = {
    addlicense = {
      command = "${lib.getExe pkgs.addlicense}";
      options = [
        "-c=Edgeless Systems GmbH"
        "-s=only"
        "-l=AGPL-3.0-only"
      ];
      includes = [
        "*.go"
        "*.nix"
        "*.sh"
      ];
    };
    vale = {
      command = "${
        pkgs.vale.withStyles (
          s: with s; [
            microsoft
            google
          ]
        )
      }/bin/vale";
      options = [ "--no-wrap" ];
      includes = [ "*.md" ];
      excludes = [
        "CODE_OF_CONDUCT.md"
        "LICENSE"
      ];
    };
  };
}
