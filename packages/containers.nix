# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  pkgs,
  writeShellApplication,
  dockerTools,
}:

let
  pushContainer =
    container:
    writeShellApplication {
      name = "push-${container.name}";
      runtimeInputs = with pkgs; [
        crane
        gzip
      ];
      text = ''
        imageName="$1"
        tmpdir=$(mktemp -d)
        trap 'rm -rf $tmpdir' EXIT
        gunzip < "${container}" > "$tmpdir/image.tar"
        crane push "$tmpdir/image.tar" "$imageName:${container.imageTag}"
      '';
    };

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
    coordinator = dockerTools.buildImage {
      name = "coordinator";
      tag = "v${pkgs.contrast.version}";
      copyToRoot =
        (with pkgs; [
          util-linux
          e2fsprogs
          coreutils
        ])
        ++ (with dockerTools; [ caCertificates ]);
      config = {
        Cmd = [ "${pkgs.contrast.coordinator}/bin/coordinator" ];
        Env = [ "PATH=/bin" ]; # This is only here for policy generation.
      };
    };

    initializer = dockerTools.buildImage {
      name = "initializer";
      tag = "v${pkgs.contrast.version}";
      copyToRoot = with dockerTools; [ caCertificates ];
      config = {
        Cmd = [ "${pkgs.contrast.initializer}/bin/initializer" ];
        Env = [ "PATH=/bin" ]; # This is only here for policy generation.
      };
    };

    openssl = dockerTools.buildImage {
      name = "openssl";
      tag = "v${pkgs.contrast.version}";
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

    cryptsetup = dockerTools.buildImage {
      name = "cryptsetup";
      tag = "v${pkgs.contrast.version}";
      copyToRoot = with pkgs; [
        busybox
        cryptsetup
        e2fsprogs # mkfs.ext4
        mount
        util-linux # blkid
        openssl
      ];
      config = {
        Cmd = [ "bash" ];
        Env = [ "PATH=/bin" ]; # This is only here for policy generation.
      };
    };

    port-forwarder = dockerTools.buildImage {
      name = "port-forwarder";
      tag = "v${pkgs.contrast.version}";
      copyToRoot = with pkgs; [
        bash
        socat
      ];
    };

    service-mesh-proxy = dockerTools.buildImage {
      name = "service-mesh-proxy";
      tag = "v${pkgs.service-mesh.version}";
      copyToRoot = with pkgs; [
        envoy
        iptables-legacy
      ];
      config = {
        Cmd = [ "${pkgs.service-mesh}/bin/service-mesh" ];
        Env = [ "PATH=/bin" ]; # This is only here for policy generation.
      };
    };

    tardev-snapshotter = dockerTools.buildImage {
      name = "tardev-snapshotter";
      tag = "v${pkgs.microsoft.tardev-snapshotter.version}";
      copyToRoot = with pkgs; [ microsoft.tardev-snapshotter ];
      config = {
        Cmd = [ "${lib.getExe pkgs.microsoft.tardev-snapshotter}" ];
      };
    };

    nydus-snapshotter = dockerTools.buildImage {
      name = "nydus-snapshotter";
      tag = "v${pkgs.nydus-snapshotter.version}";
      copyToRoot = with pkgs; [
        getconf
        nydus-snapshotter
        nydus-snapshotter.config
      ];
      config = {
        Cmd = [ "${lib.getExe pkgs.nydus-snapshotter}" ];
      };
    };

    dmesg = dockerTools.buildImage {
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
  };
in
containers
// {
  push-node-installer-microsoft =
    pushOCIDir "push-node-installer-microsoft" pkgs.microsoft.contrast-node-installer-image
      "v${pkgs.contrast.version}";
  push-node-installer-kata =
    pushOCIDir "push-node-installer-kata" pkgs.kata.contrast-node-installer-image
      "v${pkgs.contrast.version}";
}
// (lib.concatMapAttrs (name: container: { "push-${name}" = pushContainer container; }) containers)
