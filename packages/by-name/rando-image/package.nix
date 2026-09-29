{
  lib,
  nixos,
}:

nixos [
  (
    {
      config,
      lib,
      pkgs,
      modulesPath,
      ...
    }:
    {

      imports = [ "${modulesPath}/image/repart.nix" ];

      boot.loader.grub.enable = false;

      fileSystems = {
        "/" = {
                device = "/dev/mapper/root";
                fsType = "erofs";
                options = [ "ro" ];
            };
      };

      image.repart = {
        enable = true;
        name = "image";
        partitions = {
          "root" = {
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

    }
  )
]
