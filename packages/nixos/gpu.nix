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

  # nix-store-mount-hook mounts the VM's nix store into the container.
  # TODO(burgerdev): only do that for GPU containers
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
      package = lib.mkDefault (
        config.boot.kernelPackages.nvidiaPackages.mkDriver {
          # TODO: Investigate why the latest version breaks
          # GPU containers.
          version = "550.90.07";
          sha256_64bit = "sha256-Uaz1edWpiE9XOh0/Ui5/r6XnhB4iqc7AtLvq4xsLlzM=";
          sha256_aarch64 = "sha256-uJa3auRlMHr8WyacQL2MyyeebqfT7K6VU0qR7LGXFXI=";
          openSha256 = "sha256-VLmh7eH0xhEu/AK+Osb9vtqAFni+lx84P/bo4ZgCqj8=";
          settingsSha256 = "sha256-sX9dHEp9zH9t3RWp727lLCeJLo8QRAGhVb8iN6eX49g=";
          persistencedSha256 = "sha256-qe8e1Nxla7F0U88AbnOZm6cHxo57pnLCqtjdvOvq9jk=";
        }
      );
      nvidiaPersistenced = true;
    };

    # Configure the persistenced for use with CC GPUs (e.g. H100).
    # TODO(msanft): This needs to be adjusted for non-CC-GPUs.
    # See: https://docs.nvidia.com/cc-deployment-guide-snp.pdf (Page 23 & 24)
    systemd.services."nvidia-persistenced" = {
      wantedBy = [ "kata-containers.target" ];
      serviceConfig.ExecStart = lib.mkForce "${lib.getExe config.hardware.nvidia.package.persistenced} --uvm-persistence-mode --verbose";
    };

    hardware.graphics.enable = true;
    hardware.nvidia-container-toolkit.enable = true;

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

    boot.kernelParams = lib.optionals config.contrast.kata.enable [ "pci=realloc=off" ];

    services.xserver.videoDrivers = [ "nvidia" ];
  };
}
