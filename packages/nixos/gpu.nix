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
    let
      persistenced_production = config.boot.kernelPackages.nvidiaPackages.production.persistenced;
    in
    (
      (config.boot.kernelPackages.nvidiaPackages.mkDriver rec {
        url = "https://us.download.nvidia.com/tesla/${version}/NVIDIA-Linux-x86_64-${version}.run";
        version = "590.48.01";
        sha256_64bit = "sha256-ueL4BpN4FDHMh/TNKRCeEz3Oy1ClDWto1LO/LWlr1ok=";
        openSha256 = "sha256-hECHfguzwduEfPo5pCDjWE/MjtRDhINVr4b1awFdP44=";
        # Persistenced release isn't guaranteed to exist for the driver versions we are using, so follow production.
        persistencedVersion = persistenced_production.version;
        persistencedSha256 = persistenced_production.src.outputHash;
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
    systemd.services."nvidia-container-toolkit-cdi-generator" = {
      wantedBy = [ "kata-containers.target" ];
      before = [ "kata-agent.service" ];
    };
    systemd.services."kata-agent" = {
      after = [ "nvidia-container-toolkit-cdi-generator.service" ];
      requires = [ "nvidia-container-toolkit-cdi-generator.service" ];
    };

    hardware.nvidia-container-toolkit.enable = true;
    hardware.nvidia-container-toolkit.mount-nvidia-docker-1-directories = true;

    # Make NVIDIA the "default" graphics driver to replace Mesa,
    # which saves us another Perl dependency.
    hardware.graphics.package = nvidiaPackage;
    hardware.graphics.package32 = nvidiaPackage;

    services.xserver.videoDrivers = [ "nvidia" ];
  };
}
