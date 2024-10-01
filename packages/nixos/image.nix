# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  config,
  lib,
  pkgs,
  ...
}:

{
  # We build the image with systemd-repart, which integrates well
  # with the systemd utilities we use for dm-verity, UKI, etc.
  # However, we do not use the repart unit, as we don't want
  # dynamic repartitioning at run- / boot-time.
  image.repart = {
    name = "image-podvm-gpu";
    version = "1-rc1";

    # This defines the actual partition layout.
    partitions = {
      # EFI System Partition, holds the UKI.
      "00-esp" = {
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
          MakeDirectories = "/bin /boot /dev /etc /home /lib /lib64 /mnt /nix /opt /proc /root /run /srv /sys /tmp /usr /var";
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

  # TODO(msanft): fix upstream, patch available here:
  # https://github.com/edgelesssys/nixpkgs/commit/7d68972f2a145b6705b901d3ec7eebb235b7aca8?diff=split&w=1#diff-b636d2e49e098b09a6f1b276ad981f0772cc9f93b23c3d23b3bdff54cd8fb287R702-R703
  # But we don't want to use a forked nixpkgs.
  assertions = lib.mkForce [ ];
}
