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
        version = "550.144.03";
        sha256_64bit = "sha256-akg44s2ybkwOBzZ6wNO895nVa1KG9o+iAb49PduIqsQ=";
        openSha256 = "sha256-ygH9/UOWsdG53eqMbfUcyLAzAN39LJNo+uT4Wue0/7g=";
        persistencedSha256 = "sha256-pwbVQ0De8Q4L4XqV11uQIsLUUPFjL9+sABRgGGyr+wc=";
        # settingsSha256 and sha256_aarch64 are not used, but must be non-null to satisfy assertions
        # in the upstream `mkDriver` implementation.
        settingsSha256 = "";
        sha256_aarch64 = "";
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

  # nix-store-mount-hook mounts the VM's nix store into the container.
  # TODO(burgerdev): only do that for containers that actually get a GPU device.
  nix-store-mount-hook = pkgs.writeShellApplication {
    name = "nix-store-mount-hook";
    runtimeInputs = with pkgs; [
      coreutils
      util-linux
      jq
    ];
    text = ''
      # Reads from the state JSON supplied on stdin.
      bundle="$(jq -r .bundle)"
      rootfs="$bundle/rootfs"
      id="$(basename "$bundle")"

      lower=/nix/store
      target="$rootfs$lower"
      mkdir -p "$target"

      overlays="/run/kata-containers/nix-overlays/$id"
      upperdir="$overlays/upperdir"
      workdir="$overlays/workdir"
      mkdir -p "$upperdir" "$workdir"

      mount -t overlay -o "lowerdir=$lower:$target,upperdir=$upperdir,workdir=$workdir" none "$target"
    '';
  };
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

    # WARNING: Kata sets systemd's default target to `kata-containers.target`. Thus, some upstream services may not work out-of-the-box,
    # as they are `WantedBy=multi-user.target` or similar. In such cases, the service needs to be adjusted to be `WantedBy=kata-containers.target`
    # instead.

    # Configure the persistenced for use with CC GPUs (e.g. H100).
    # See: https://docs.nvidia.com/cc-deployment-guide-snp.pdf (Page 23 & 24)
    # Note that the current setup does not support all use cases with non-CC-GPUs.
    # See: https://github.com/NVIDIA/open-gpu-kernel-modules/issues/531#issuecomment-1711195538
    # and https://github.com/AMDESE/AMDSEV/issues/174#issuecomment-1660103561
    systemd.services."nvidia-persistenced" = {
      wantedBy = [ "kata-containers.target" ];
      serviceConfig.ExecStart = lib.mkForce "${lib.getExe config.hardware.nvidia.package.persistenced} --uvm-persistence-mode --verbose";
    };

    # kata-containers.target needs to pull this in so that we get a valid
    # CDI configuration inside the PodVM. This is not necessary, as we use the
    # legacy mode as of now, but will be once we switch to CDI.
    systemd.services."nvidia-container-toolkit-cdi-generator".wantedBy = [ "kata-containers.target" ];

    hardware.nvidia-container-toolkit.enable = true;

    # Make NVIDIA the "default" graphics driver to replace Mesa,
    # which saves us another Perl dependency.
    hardware.graphics.package = nvidiaPackage;
    hardware.graphics.package32 = nvidiaPackage;

    image.repart.partitions."10-root".contents = {
      "/usr/share/oci/hooks/prestart/nvidia-container-toolkit.sh".source =
        lib.getExe pkgs.nvidia-ctk-oci-hook;
      "/usr/share/oci/hooks/prestart/nix-store-mount-hook.sh".source = lib.getExe nix-store-mount-hook;
    };

    environment.systemPackages = [ pkgs.nvidia-ctk-with-config ];

    boot.initrd.kernelModules = [
      # Extra kernel modules required to talk to the GPU in CC-Mode.
      "ecdsa_generic"
      "ecdh"
    ];

    services.xserver.videoDrivers = [ "nvidia" ];
  };
}
