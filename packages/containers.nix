# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  pkgs,
  contrastPkgs,
  writeShellApplication,
  dockerTools,
}:

let
  pushOCIDir =
    name: dir: tag:
    writeShellApplication {
      name = "push-${name}";
      runtimeInputs = with pkgs; [ crane ];
      text = ''
        imageName="$1"
        crane push "${dir}" "$imageName:${tag}"
      '';
    };

  containers = {
    coordinator = contrastPkgs.buildOciImage {
      name = "coordinator";
      tag = "v${contrastPkgs.contrast.coordinator.version}";
      copyToRoot =
        (with pkgs; [
          busybox
          e2fsprogs # mkfs.ext4
          libuuid # blkid
          iptables-legacy
        ])
        ++ (with dockerTools; [ caCertificates ]);
      config = {
        Cmd = [ "${contrastPkgs.contrast.coordinator}/bin/coordinator" ];
        Env = [
          "PATH=/bin" # Explicitly setting this prevents containerd from setting a default PATH.
          "XTABLES_LOCKFILE=/dev/shm/xtables.lock" # Tells iptables where to create the lock file, since the default path does not exist in our image.
        ];
      };
    };

    initializer = contrastPkgs.buildOciImage {
      name = "initializer";
      tag = "v${contrastPkgs.contrast.initializer.version}";
      copyToRoot =
        (with pkgs; [
          busybox
          cryptsetup
          e2fsprogs # mkfs.ext4
          libuuid # blkid
          iptables-legacy
        ])
        ++ (with dockerTools; [ caCertificates ]);
      config = {
        # Use Entrypoint so we can append arguments.
        Entrypoint = [ "${contrastPkgs.contrast.initializer}/bin/initializer" ];
        Env = [
          "PATH=/bin" # Explicitly setting this prevents containerd from setting a default PATH.
          "XTABLES_LOCKFILE=/dev/shm/xtables.lock" # Tells iptables where to create the lock file, since the default path does not exist in our image.
        ];
      };
    };

    openssl = contrastPkgs.buildOciImage {
      name = "openssl";
      tag = "v${contrastPkgs.contrast.cli.version}";
      copyToRoot = with pkgs; [
        busybox
        openssl
        curlMinimal
      ];
      config = {
        Cmd = [ "bash" ];
        Env = [ "PATH=/bin" ]; # This is only here for policy generation.
      };
    };

    port-forwarder = contrastPkgs.buildOciImage {
      name = "port-forwarder";
      tag = "v${contrastPkgs.contrast.cli.version}";
      copyToRoot = with pkgs; [
        bash
        socat
      ];
    };

    service-mesh-proxy = contrastPkgs.buildOciImage {
      name = "service-mesh-proxy";
      tag = "v${contrastPkgs.service-mesh.version}";
      copyToRoot = with pkgs; [
        busybox
        envoy-bin
        iptables-legacy
      ];
      config = {
        # Use Entrypoint so we can append arguments.
        Entrypoint = [ "${contrastPkgs.service-mesh}/bin/service-mesh" ];
        Env = [
          "PATH=/bin"
          "XTABLES_LOCKFILE=/dev/shm/xtables.lock" # Tells iptables where to create the lock file, since the default path does not exist in our image.
        ];
      };
    };

    dmesg = contrastPkgs.buildOciImage {
      name = "dmesg";
      tag = "v0.0.1";
      copyToRoot = with pkgs; [
        busybox
        libuuid
      ];
      config = {
        Cmd = [
          "sh"
          "-c"
          "mknod /dev/kmsg c 1 11 && dmesg --follow --color=always --nopager"
        ];
        Env = [ "PATH=/bin" ]; # This is only here for policy generation.
      };
    };

    cleanup-bare-metal = contrastPkgs.buildOciImage {
      name = "cleanup-bare-metal";
      tag = "latest";
      copyToRoot =
        (with pkgs; [
          cacert
          busybox
        ])
        ++ (with contrastPkgs.scripts; [
          cleanup-bare-metal
          cleanup-namespaces
          cleanup-containerd
          nix-gc
        ]);
      config = {
        Cmd = [ "cleanup-bare-metal" ];
      };
    };

    memdump = contrastPkgs.buildOciImage {
      name = "memdump";
      tag = "latest";
      copyToRoot = with pkgs; [
        busybox
        socat
        gdb
        jq
      ];
    };

    debugshell = contrastPkgs.buildOciImage {
      name = "debugshell";
      tag = contrastPkgs.contrast.contrast.version;
      copyToRoot = with pkgs; [
        busybox
        bash
        coreutils
        ncurses
        contrastPkgs.debugshell
        openssh
        contrastPkgs.tdx-tools
      ];
      config = {
        Entrypoint = [ "/bin/debugshell" ];
        Cmd = [ "journalctl --no-tail --no-pager -f" ];
      };
    };

    k8s-log-collector = contrastPkgs.buildOciImage {
      name = "k8s-log-collector";
      tag = "0.1.0";
      copyToRoot = with pkgs; [
        # Used when execing into the container to collect logs.
        bash
        coreutils
        gnutar
        gzip
      ];
      config = {
        Cmd = [ "${lib.getExe contrastPkgs.k8s-log-collector}" ];
        Volumes."/logs" = { };
      };
    };
  };
in
containers
// {
  push-node-installer-kata =
    pushOCIDir "push-node-installer-kata" contrastPkgs.contrast.node-installer-image
      "v${contrastPkgs.contrast.nodeinstaller.version}";
  push-node-installer-kata-gpu =
    pushOCIDir "push-node-installer-kata-gpu" contrastPkgs.contrast.node-installer-image.gpu
      "v${contrastPkgs.contrast.nodeinstaller.version}";
}
// (lib.concatMapAttrs (name: container: {
  "push-${name}" = pushOCIDir name container.outPath container.meta.tag;
}) containers)
