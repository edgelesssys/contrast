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
    nixf-diagnose = {
      enable = true;
      ignore = [ "sema-primop-overridden" ];
    };
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
    # keep-sorted start block=true
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
    ascii-lint = {
      command = "${lib.getExe pkgs.contrastPkgs.scripts.ascii-lint}";
      includes = [
        "docs/docs/*.md"
        "README.md"
      ];
    };
    # Make sure that every markdown page is listed in the sidebar, and vice-versa.
    check-sidebar = {
      command = "${lib.getExe pkgs.contrastPkgs.scripts.check-sidebar}";
      includes = [ "docs/sidebars.js" ];
    };
    # Docs shouldn't use links containing the full URL, but rather relative links.
    # Otherwise the links won't point to the correct versioned location after a release.
    docs-selfref-lint = {
      command = "${lib.getExe pkgs.contrastPkgs.scripts.docs-selfref-lint}";
      includes = [ "docs/docs/*.md" ];
    };
    github-matrix-parser = {
      command = "${lib.getExe pkgs.contrastPkgs.github-matrix-parser}";
      options = [ "--check" ];
      includes = [ ".github/workflows/*.yml" ];
    };
    # Catch debug arguments in nix code that were accidentally left on true.
    lint-no-debug = {
      command = "${lib.getExe pkgs.contrastPkgs.scripts.lint-no-debug}";
      includes = [ "*.nix" ];
    };
    lychee-internal-links = {
      command = "${lib.getExe pkgs.lychee}";
      options = [
        "--config"
        "tools/lychee/config-internal.toml"
      ];
      includes = [
        "*.md"
        "*.html"
      ];
    };
    # We need to provide mdsh with the tools we use inside the markdown code.
    mdsh = {
      command = "${lib.getExe pkgs.contrastPkgs.scripts.mdsh-fmt}";
      includes = [
        "docs/docs/*.md"
        "docs/docs/**/*.md"
      ];
    };
    renovate = {
      command = "${lib.getExe' pkgs.renovate "renovate-config-validator"}";
      options = [ "--strict" ];
      includes = [ "renovate.json5" ];
    };
    stylua = {
      command = "${lib.getExe pkgs.stylua}";
      includes = [ "*.lua" ];
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
        "dev-docs/e2e/well-known-errors.md"
      ];
    };
    # keep-sorted end
  };
}
