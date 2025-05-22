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
        "-l=AGPL-3.0-only"
      ];
      includes = [
        "*.go"
        "*.nix"
        "*.sh"
      ];
      excludes = [ "enterprise/**" ];
    };
    addlicense-enterprise = {
      command = "${lib.getExe pkgs.addlicense}";
      options = [
        "-c=Edgeless Systems GmbH"
        "-s=only"
        "-l=BUSL-1.1"
      ];
      includes = [
        "enterprise/**/*.go"
        "enterprise/**/*.nix"
        "enterprise/**/*.sh"
      ];
    };
    # Build must not be used in the enterprise directory.
    # They are only needed in non-enterprise code to switch between
    # enterprise and non-enterprise implementation.
    # e2e test shouldn't use enterprise code at all.
    lint-buildtags = {
      command = "${lib.getExe pkgs.scripts.lint-buildtags}";
      options = [ "enterprise" ];
      includes = [
        "enterprise/**/*.go"
        "e2e/**/*.go"
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
