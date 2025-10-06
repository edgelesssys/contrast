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
        version = "580.95.05";
        sha256_64bit = "sha256-hJ7w746EK5gGss3p8RwTA9VPGpp2lGfk5dlhsv4Rgqc=";
        openSha256 = "sha256-RFwDGQOi9jVngVONCOB5m/IYKZIeGEle7h0+0yGnBEI=";
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

  nividia-container-toolkit-config = (pkgs.formats.toml { }).generate "config.toml" {
    disable-require = true;
    supported-driver-capabilities = "compat32,compute,display,graphics,ngx,utility,video";
    nvidia-container-cli = {
      no-pivot = true;
      debug = "/var/log/nvidia-kata-container/nvidia-container-toolkit.log";
      environment = [ ];
      ldcache = "/tmp/ld.so.cache";
      ldconfig = "@${lib.getBin pkgs.glibc}/bin/ldconfig";
      load-kmods = true;
      no-cgroups = true;
      path = lib.getExe' pkgs.libnvidia-container "nvidia-container-cli";
    };
    nvidia-container-runtime = {
      debug = "/var/log/nvidia-kata-container/nvidia-container-runtime.log";
      log-level = "debug";
      mode = "cdi";
      runtimes = [
        "docker-runc"
        "runc"
        "crun"
      ];
      modes = {
        cdi = {
          annotation-prefixes = [ "cdi.k8s.io/" ];
          default-kind = "nvidia.com/gpu";
          spec-dirs = [ "/var/run/cdi" ];
        };
        csv.mount-spec-path = "/etc/nvidia-container-runtime/host-files-for-container.d";
      };
    };
    nvidia-container-runtime-hook = {
      path = lib.getExe' pkgs.nvidia-container-toolkit.tools "nvidia-container-runtime-hook";
      skip-mode-detection = true;
    };
    nvidia-ctk.path = lib.getExe' pkgs.nvidia-container-toolkit "nvidia-ctk";
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

    image.repart.partitions."10-root" = {
      contents = {
        "/usr/share/oci/hooks/prestart/nvidia-container-toolkit.sh".source = lib.getExe (
          pkgs.writeShellApplication {
            name = "nvidia-ctk-oci-hook";
            text = ''
              ${lib.getExe' pkgs.nvidia-container-toolkit.tools "nvidia-container-runtime-hook"} \
                -config ${nividia-container-toolkit-config} \
                -debug "$@" > /var/log/nvidia-hook.log 2>&1
            '';
          }
        );
        "/usr/share/oci/hooks/prestart/nix-store-mount-hook.sh".source = lib.getExe nix-store-mount-hook;
      };
      # Storepaths must include the closure of contents, otherwise it is not included.
      storePaths = [
        nividia-container-toolkit-config
        nix-store-mount-hook
        pkgs.nvidia-container-toolkit.tools
      ];
    };

    services.xserver.videoDrivers = [ "nvidia" ];
  };
}
