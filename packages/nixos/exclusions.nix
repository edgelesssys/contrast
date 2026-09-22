# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  config,
  lib,
  modulesPath,
  pkgs,
  ...
}:

let
  # Some packages are assumed to be required for all NixOS instances and can't simply be disabled.
  # These are specified in the corePackageNames field here
  # https://github.com/NixOS/nixpkgs/blob/7c5431054d2455fd984c6242e572cfbf8031dce5/nixos/modules/config/system-path.nix#L11-L41
  # which we can't directly overwrite. Copying the list here would be brittle and hard to maintain,
  # therefore we parse the module here, filter the corePackages list and set the filtered list with
  # mkForce.
  #
  # A nice side-effect of this approach is that it also removes openssh, which is added here:
  # https://github.com/NixOS/nixpkgs/blob/9b65ebe9e703d6c3e8d83c51444c5bacc04568f6/nixos/modules/programs/ssh.nix#L338.
  systemPathRaw = import (modulesPath + "/config/system-path.nix") {
    inherit config lib pkgs;
  };

  excludes = [
    "curl"
    "netcat"
  ];

  filteredCorePackages = lib.filter (
    p: !(builtins.elem (p.pname or p.name or "") excludes)
  ) systemPathRaw.config.environment.corePackages;
in
{
  config = lib.mkIf (!config.contrast.debug.enable) {
    environment.corePackages = lib.mkForce filteredCorePackages;

    system.forbiddenDependenciesRegexes = [
      "curl"
      "openssh"
      "netcat"
      "libxslt"
      "libxml2"
    ];
  };
}
