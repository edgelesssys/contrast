# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

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
    zizmor.enable = true;
    zizmor.includes = [
      ".github/workflows/*.yml"
      ".github/workflows/*.yaml"
      ".github/actions/**/*.yml"
      ".github/actions/**/*.yaml"
    ];
    # keep-sorted end
  };
  settings.formatter = {
    # Notice! addlicense won't enforce the correct license header
    # but only check if there is something like a license header at all.
    # It will neither fail on nor update incorrect license headers.
    addlicense = {
      command = "${lib.getExe pkgs.addlicense}";
      options = [
        "-c=Edgeless Systems GmbH"
        "-s=only"
        "-l=BUSL-1.1"
      ];
      includes = [
        "*.go"
        "*.nix"
        "*.sh"
      ];
    };
    # Catch debug arguments in nix code that were accidentally left on true.
    lint-no-debug = {
      command = "${lib.getExe pkgs.scripts.lint-no-debug}";
      includes = [ "*.nix" ];
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
