# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  config,
  lib,
  ...
}:

{
  boot.loader.grub.enable = false;
  boot.kernelParams = [
    "systemd.verity=yes"
    "selinux=0"
  ];
  boot.supportedFilesystems = [
    "erofs"
    "vfat"
  ];
  boot.initrd = {
    supportedFilesystems = [
      "erofs"
      "vfat"
    ];
    availableKernelModules = [
      "dm_mod"
      "dm_verity"
      "overlay"
    ];
    services.lvm.enable = true; # For additional udev rules needed by dm-verity.
    systemd = {
      enable = true;
      additionalUpstreamUnits = [
        "veritysetup-pre.target"
        "veritysetup.target"
        "remote-veritysetup.target"
      ];
      storePaths = [
        "${config.boot.initrd.systemd.package}/lib/systemd/systemd-veritysetup"
        "${config.boot.initrd.systemd.package}/lib/systemd/system-generators/systemd-veritysetup-generator"
      ];
    };
  };

  fileSystems = {
    "/" = {
      device = "/dev/mapper/root";
      fsType = "erofs";
      options = [ "ro" ];
    };
  }
  # Create tmpfs on directories that need to be writable for activation.
  # TODO(msanft): This needs better support upstream.
  // lib.listToAttrs (
    lib.map
      (path: {
        name = path;
        value = {
          fsType = "tmpfs";
          neededForBoot = true;
        };
      })
      [
        "/var"
        "/bin"
        "/usr/bin"
        "/tmp"
        "/lib"
        "/root"
        "/lib64"
      ]
  );
  # If /etc is read-only, we need to provide the machine-id file as a mount point for systemd.
  # https://www.freedesktop.org/software/systemd/man/256/machine-id.html#Initialization
  environment.etc."machine-id".text = "";

  networking.firewall.enable = false;

  # We do not require dynamic host configuration.
  # Additionally, dhcpcd could allow for e.g. route manipulation from the host.
  networking.dhcpcd.enable = false;

  # Images are immutable, so no need to include Nix.
  nix.enable = false;

  # Interpreter-less activation bits, tailored to our needs:
  # Source: https://github.com/NixOS/nixpkgs/blob/a4741ea333f97cca0680d1eb485907f0e4a0eb3a/nixos/modules/profiles/perlless.nix
  # We do not include the upstream module as-is, as we don't need sophisticated user generation, for example.
  #
  # Remove perl from activation
  system.etc.overlay = {
    enable = true;
    mutable = false;
  };
  # simple replacement for update-users-groups.pl
  systemd.sysusers.enable = true;
  # Random perl remnants
  system.disableInstallerTools = true;
  programs.less.lessopen = null;
  programs.command-not-found.enable = false;
  boot.enableContainers = false;
  environment.defaultPackages = [ ];
  documentation.enable = false;
  # Check that the system does not contain a Nix store path that contains the
  # string "perl" or "python".
  system.forbiddenDependenciesRegexes = [
    "perl"
  ]
  ++ lib.optionals (!config.contrast.debug.enable) [
    "python" # Some of the debug packages need Python.
  ];

  nixpkgs.hostPlatform.system = "x86_64-linux";
  system.switch.enable = false;
  system.stateVersion = "24.05";
}
