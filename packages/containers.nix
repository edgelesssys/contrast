# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  pkgs,
  stdenvNoCC,
  writeShellApplication,
  dockerTools,
}:

let
  toOciImage =
    docker-tarball:
    stdenvNoCC.mkDerivation {
      name = "${lib.strings.removeSuffix ".tar.gz" docker-tarball.name}";
      src = docker-tarball;
      dontUnpack = true;
      nativeBuildInputs = with pkgs; [ skopeo ];
      buildPhase = ''
        runHook preBuild
        skopeo copy docker-archive:$src oci:$out --insecure-policy --tmpdir .
        runHook postBuild
      '';
      meta = {
        tag = docker-tarball.imageTag;
      };
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

  containers = lib.mapAttrs (_name: toOciImage) {
    coordinator = dockerTools.buildImage {
      name = "coordinator";
      tag = "v${pkgs.contrast.version}";
      copyToRoot =
        (with pkgs; [
          busybox
          e2fsprogs # mkfs.ext4
          libuuid # blkid
          iptables-legacy
        ])
        ++ (with dockerTools; [ caCertificates ]);
      config = {
        Cmd = [ "${pkgs.contrast.coordinator}/bin/coordinator" ];
        Env = [
          "PATH=/bin" # Explicitly setting this prevents containerd from setting a default PATH.
          "XTABLES_LOCKFILE=/dev/shm/xtables.lock" # Tells iptables where to create the lock file, since the default path does not exist in our image.
        ];
      };
    };

    initializer = dockerTools.buildImage {
      name = "initializer";
      tag = "v${pkgs.contrast.version}";
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
        Entrypoint = [ "${pkgs.contrast.initializer}/bin/initializer" ];
        Env = [
          "PATH=/bin" # Explicitly setting this prevents containerd from setting a default PATH.
          "XTABLES_LOCKFILE=/dev/shm/xtables.lock" # Tells iptables where to create the lock file, since the default path does not exist in our image.
        ];
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
        busybox
        envoy
        iptables-legacy
      ];
      config = {
        # Use Entrypoint so we can append arguments.
        Entrypoint = [ "${pkgs.service-mesh}/bin/service-mesh" ];
        Env = [
          "PATH=/bin"
          "XTABLES_LOCKFILE=/dev/shm/xtables.lock" # Tells iptables where to create the lock file, since the default path does not exist in our image.
        ];
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

    cleanup-bare-metal = dockerTools.buildImage {
      name = "cleanup-bare-metal";
      tag = "latest";
      copyToRoot = with pkgs; [
        cacert
        busybox
        scripts.cleanup-bare-metal
        scripts.cleanup-namespaces
        scripts.cleanup-containerd
        scripts.nix-gc
      ];
      config = {
        Cmd = [ "cleanup-bare-metal" ];
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
  push-node-installer-kata-gpu =
    pushOCIDir "push-node-installer-kata-gpu" pkgs.kata.contrast-node-installer-image.gpu
      "v${pkgs.contrast.version}";
}
// (lib.concatMapAttrs (name: container: {
  "push-${name}" = pushOCIDir name container.outPath container.meta.tag;
}) containers)
