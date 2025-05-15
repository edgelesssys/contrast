# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  config,
  pkgs,
  lib,
  ...
}:

let
  cfg = config.contrast.image;
in

{
  options.contrast.image = {
    microVM = lib.mkEnableOption "Build a micro VM image";
  };

  config = {
    system.image.version = "1-rc1";

    documentation = {
      man.enable = false;
      nixos.enable = false;
    };

    # We build the image with systemd-repart, which integrates well
    # with the systemd utilities we use for dm-verity, UKI, etc.
    # However, we do not use the repart unit, as we don't want
    # dynamic repartitioning at run- / boot-time.
    image.repart = {
      name = "image-podvm-gpu";
      inherit (config.system.image) version;

      mkfsOptions.erofs = [
        # Dereference hardlinks to avoid different number of inodes between images
        # built on systems with nix store optimization enabled and disabled.
        "--hard-dereference"
      ];

      # This defines the actual partition layout.
      partitions = {
        # EFI System Partition, holds the UKI.
        # Only build this partition if we need a bootable image (i.e. not a micro VM).
        "00-esp" = lib.mkIf (!cfg.microVM) {
          contents = {
            "/".source = pkgs.runCommand "esp-contents" { } ''
              mkdir -p $out/EFI/BOOT
              cp ${config.system.build.uki}/${config.system.boot.loader.ukiFile} $out/EFI/BOOT/BOOTX64.EFI
            '';
          };
          repartConfig = {
            Type = "esp";
            Format = "vfat";
            SizeMinBytes = "64M";
            UUID = "null"; # Fix partition UUID for reproducibility.
          };
        };

        # Root filesystem.
        "10-root" = {
          contents = {
            "/pause_bundle".source = "${pkgs.pause-bundle}/pause_bundle";
          };
          storePaths = [ config.system.build.toplevel ];
          repartConfig = {
            Type = "root";
            Format = "erofs";
            Label = "root";
            Verity = "data";
            VerityMatchKey = "root";
            Minimize = "best";
            # We need to ensure that mountpoints are available.
            # TODO (Maybe): This could be done more elegantly with CopyFiles and a skeleton tree in the vcs.
            MakeDirectories = "/bin /boot /dev /etc /home /lib /lib64 /mnt /nix /opt /proc /root /run /srv /sys /tmp /usr/bin /var";
          };
        };

        # Verity hashes for the root filesystem.
        "20-root-verity" = {
          repartConfig = {
            Type = "root-verity";
            Label = "root-verity";
            Verity = "hash";
            VerityMatchKey = "root";
            Minimize = "best";
          };
        };
      };
    };
  };
}
