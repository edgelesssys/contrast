# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  pkgs,
  contrastPkgs,
}:

let
  contrastVersion = contrastPkgs.contrast.base.cli.version;

  containers = {
    openssl = contrastPkgs.buildOciImage {
      name = "openssl";
      tag = "v${contrastVersion}";
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
      tag = "v${contrastVersion}";
      copyToRoot = with pkgs; [
        bash
        socat
      ];
    };

    service-mesh-proxy = contrastPkgs.buildOciImage {
      name = "service-mesh-proxy";
      tag = "v${contrastVersion}";
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
      tag = contrastVersion;
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
// (lib.concatMapAttrs (name: container: {
  "push-${name}" = contrastPkgs.pushOCIDir name container.outPath container.meta.tag;
}) containers)
