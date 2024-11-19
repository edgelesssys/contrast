# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

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

  fileSystems =
    {
      "/" = {
        device = "/dev/mapper/root";
        fsType = "erofs";
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
          "/etc"
          "/bin"
          "/usr/bin"
          "/tmp"
          "/lib"
          "/root"
          "/lib64"
        ]
    );

  # We cant remount anything in the userspace, as we already
  # have the rootfs mounted read-only from the initrd.
  systemd.suppressedSystemUnits = [ "systemd-remount-fs.service" ];

  networking.firewall.enable = false;

  nixpkgs.hostPlatform.system = "x86_64-linux";
  system.switch.enable = false;
  system.stateVersion = "24.05";
}
