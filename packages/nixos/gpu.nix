# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

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
      (config.boot.kernelPackages.nvidiaPackages.mkDriver rec {
        url = "https://us.download.nvidia.com/tesla/${version}/NVIDIA-Linux-x86_64-${version}.run";
        version = "570.158.01";
        sha256_64bit = "sha256-R/TMmtB79xjQ3Ojb/ABFzTas7WmXEsfoQX+F+4fZGOs=";
        openSha256 = "sha256-hScINXiAuJWhuO4oLDAiVGZS97OXDGr1IiFZxkGidfg=";
        persistencedVersion = "570.169"; # There is no persistenced release for 570.158.01, use current production version.
        persistencedSha256 = "sha256-dttFu+TmbFI+mt1MbbmJcUnc1KIJ20eHZDR7YzfWmgE=";
        useSettings = false;
      }).override
      {
        disable32Bit = true;
      }
    ).overrideAttrs
      (_oldAttrs: {
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

    environment.systemPackages = [
      pkgs.nvidia-ctk-with-config
      pkgs.nvidia-ctk-with-config.tools
    ];

    boot.initrd.kernelModules = [
      # Extra kernel modules required to talk to the GPU in CC-Mode.
      "ecdsa_generic"
      "ecdh"
    ];

    services.xserver.videoDrivers = [ "nvidia" ];
  };
}
