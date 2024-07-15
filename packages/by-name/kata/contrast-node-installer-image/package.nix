# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  ociLayerTar,
  ociImageManifest,
  ociImageLayout,
  contrast-node-installer,
  kata,
  pkgsStatic,
  writers,
}:

let
  node-installer = ociLayerTar {
    files = [
      {
        source = lib.getExe contrast-node-installer;
        destination = "/bin/node-installer";
      }
      {
        source = "${pkgsStatic.util-linux}/bin/nsenter";
        destination = "/bin/nsenter";
      }
    ];
  };

  launch-digest = lib.removeSuffix "\n" (
    builtins.readFile "${kata.runtime-class-files}/launch-digest.hex"
  );
  runtime-handler = lib.removeSuffix "\n" (
    builtins.readFile "${kata.runtime-class-files}/runtime-handler"
  );

  installer-config = ociLayerTar {
    files = [
      {
        source = writers.writeJSON "contrast-node-install.json" {
          files = [
            {
              url = "file:///opt/edgeless/share/kata-containers.img";
              path = "/opt/edgeless/${runtime-handler}/share/kata-containers.img";
            }
            {
              url = "file:///opt/edgeless/share/kata-kernel";
              path = "/opt/edgeless/${runtime-handler}/share/kata-kernel";
            }
            {
              url = "file:///opt/edgeless/bin/qemu-system-x86_64";
              path = "/opt/edgeless/${runtime-handler}/bin/qemu-system-x86_64";
            }
            {
              url = "file:///opt/edgeless/share/OVMF.fd";
              path = "/opt/edgeless/${runtime-handler}/share/OVMF.fd";
            }
            {
              url = "file:///opt/edgeless/bin/containerd-shim-contrast-cc-v2";
              path = "/opt/edgeless/${runtime-handler}/bin/containerd-shim-contrast-cc-v2";
            }
            {
              url = "file:///opt/edgeless/bin/kata-runtime";
              path = "/opt/edgeless/${runtime-handler}/bin/kata-runtime";
            }
            {
              url = "file:///opt/edgeless/share/qemu/kvmvapic.bin";
              path = "/opt/edgeless/${runtime-handler}/share/qemu/kvmvapic.bin";
            }
            {
              url = "file:///opt/edgeless/share/qemu/linuxboot_dma.bin";
              path = "/opt/edgeless/${runtime-handler}/share/qemu/linuxboot_dma.bin";
            }
            {
              url = "file:///opt/edgeless/share/qemu/efi-virtio.rom";
              path = "/opt/edgeless/${runtime-handler}/share/qemu/efi-virtio.rom";
            }
          ];
          runtimeHandlerName = runtime-handler;
          inherit (kata.runtime-class-files) debugRuntime;
        };
        destination = "/config/contrast-node-install.json";
      }
    ];
  };

  kata-container-img = ociLayerTar {
    files = [
      {
        source = kata.runtime-class-files.image;
        destination = "/opt/edgeless/share/kata-containers.img";
      }
      {
        source = kata.runtime-class-files.kernel;
        destination = "/opt/edgeless/share/kata-kernel";
      }
    ];
  };

  ovmf = ociLayerTar {
    files = [
      {
        source = kata.runtime-class-files.ovmf;
        destination = "/opt/edgeless/share/OVMF.fd";
      }
    ];
  };

  qemu = ociLayerTar {
    files = [
      {
        source = kata.runtime-class-files.qemu-bin;
        destination = "/opt/edgeless/bin/qemu-system-x86_64";
      }
      {
        source = "${kata.runtime-class-files.qemu-share}/kvmvapic.bin";
        destination = "/opt/edgeless/share/qemu/kvmvapic.bin";
      }
      {
        source = "${kata.runtime-class-files.qemu-share}/linuxboot_dma.bin";
        destination = "/opt/edgeless/share/qemu/linuxboot_dma.bin";
      }
      {
        source = "${kata.runtime-class-files.qemu-share}/efi-virtio.rom";
        destination = "/opt/edgeless/share/qemu/efi-virtio.rom";
      }
    ];
  };

  kata-runtime = ociLayerTar {
    files = [
      {
        source = kata.runtime-class-files.kata-runtime;
        destination = "/opt/edgeless/bin/kata-runtime";
      }
      {
        source = kata.runtime-class-files.containerd-shim-contrast-cc-v2;
        destination = "/opt/edgeless/bin/containerd-shim-contrast-cc-v2";
      }
    ];
  };

  manifest = ociImageManifest {
    layers = [
      node-installer
      installer-config
      kata-container-img
      ovmf
      qemu
      kata-runtime
    ];
    extraConfig = {
      "config" = {
        "Env" = [
          "PATH=/bin:/usr/bin"
          "CONFIG_DIR=/config"
          "HOST_MOUNT=/host"
        ];
        "Entrypoint" = [ "/bin/node-installer" ];
      };
    };
    extraManifest = {
      "annotations" = {
        "org.opencontainers.image.title" = "contrast-node-installer-kata";
        "org.opencontainers.image.description" = "Contrast Node Installer (Kata)";
        "systems.edgeless.contrast.snp-launch-digest" = launch-digest;
      };
    };
  };
in

ociImageLayout { manifests = [ manifest ]; }
