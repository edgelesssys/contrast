# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  config,
  lib,
  pkgs,
  ...
}:

let
  cfg = config.contrast.gpu;

  nvidiaPackage =
    (
      (config.boot.kernelPackages.nvidiaPackages.mkDriver {
        # TODO(msanft): Investigate why the latest version breaks GPU containers.
        version = "550.90.07";
        sha256_64bit = "sha256-Uaz1edWpiE9XOh0/Ui5/r6XnhB4iqc7AtLvq4xsLlzM=";
        sha256_aarch64 = "sha256-uJa3auRlMHr8WyacQL2MyyeebqfT7K6VU0qR7LGXFXI=";
        openSha256 = "sha256-VLmh7eH0xhEu/AK+Osb9vtqAFni+lx84P/bo4ZgCqj8=";
        settingsSha256 = "sha256-sX9dHEp9zH9t3RWp727lLCeJLo8QRAGhVb8iN6eX49g=";
        persistencedSha256 = "sha256-qe8e1Nxla7F0U88AbnOZm6cHxo57pnLCqtjdvOvq9jk=";
      }).override
      {
        disable32Bit = true;
      }
    ).overrideAttrs
      (oldAttrs: {
        # We strip the driver package from its dependencies on desktop software like Wayland and X11.
        # For server use-cases, we shouldn't need these. The Mesa (and thus Perl) and libGL dependencies are dropped
        # too, as GPU workloads will likely be AI-related and not graphical. The libdrm dependency is dropped as well,
        # as we're probably not going to be watching Netflix on the servers.
        # Source: https://github.com/NixOS/nixpkgs/blob/eac1633a086e8e109e00ce58c0b47721da1dbdfd/pkgs/os-specific/linux/nvidia-x11/generic.nix#L100C3-L114C6
        libPath = lib.makeLibraryPath (
          with pkgs;
          [
            zlib
            stdenv.cc.cc
            openssl
            dbus # for nvidia-powerd
          ]
        );

        # Hack to pass the "right" (i.e. the overridden) version of the nvidia driver to the persistenced.
        # Looking at the package definition, it _should_ already do so, but it doesn't.
        # So for now, override all occurences of `nvidia_x11` in the persistenced package "manually".
        # We can't do an `override` on persistenced itself unfortunately, as it's call site doesn't allow this:
        # https://github.com/NixOS/nixpkgs/blob/4d2418ebbfb107485b44aaa1b2909409322d9061/pkgs/os-specific/linux/nvidia-x11/generic.nix#L260
        # TODO(msanft): Clarify with upstream why that is the case.
        passthru = oldAttrs.passthru // {
          persistenced = oldAttrs.passthru.persistenced.overrideAttrs (oldAttrs: {
            inherit (nvidiaPackage) version makeFlags;
            src = oldAttrs.src // {
              rev = nvidiaPackage.version;
            };

            postFixup = ''
              # Save a copy of persistenced for mounting in containers
              mkdir $out/origBin
              cp $out/{bin,origBin}/nvidia-persistenced
              patchelf --set-interpreter /lib64/ld-linux-x86-64.so.2 $out/origBin/nvidia-persistenced

              patchelf --set-rpath "$(patchelf --print-rpath $out/bin/nvidia-persistenced):${nvidiaPackage}/lib" \
                $out/bin/nvidia-persistenced
            '';

            meta = oldAttrs.meta // {
              inherit (nvidiaPackage.meta) platforms;
            };
          });
        };
      });
in

{
  options.contrast.gpu = {
    enable = lib.mkEnableOption "Enable GPU support";
  };

  config = lib.mkIf cfg.enable {
    hardware.nvidia = {
      open = true;
      package = nvidiaPackage;
      nvidiaPersistenced = true;
      # Disable NVIDIA's GUI settings tool.
      nvidiaSettings = false;
      # We don't need video acceleration on a server. Disabling this
      # saves quite some disk space.
      videoAcceleration = false;
    };

    hardware.graphics.enable = true;
    hardware.nvidia-container-toolkit.enable = true;

    # Make NVIDIA the "default" graphics driver to replace Mesa,
    # which saves us another Perl dependency.
    hardware.graphics.package = nvidiaPackage;
    hardware.graphics.package32 = nvidiaPackage;

    image.repart.partitions."10-root".contents."/usr/share/oci/hooks/prestart/nvidia-container-toolkit.sh".source =
      lib.getExe pkgs.nvidia-ctk-oci-hook;

    boot.initrd.kernelModules = [
      # Extra kernel modules required to talk to the GPU in CC-Mode.
      "ecdsa_generic"
      "ecdh"
    ];

    boot.kernelParams = lib.optionals config.contrast.kata.enable [ "pci=realloc=off" ];

    services.xserver.videoDrivers = [ "nvidia" ];
  };
}
