# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  ociLayerTar,
  ociImageManifest,
  ociImageLayout,
  writers,
  hashDirs,

  contrast,
  microsoft,
  pkgsStatic,

  debugRuntime ? false,
}:

let
  node-installer = ociLayerTar {
    files = [
      {
        source = "${contrast.nodeinstaller}/bin/node-installer";
        destination = "/bin/node-installer";
      }
      {
        source = "${pkgsStatic.util-linux}/bin/nsenter";
        destination = "/bin/nsenter";
      }
    ];
  };

  installer-config = ociLayerTar {
    files = [
      {
        source = writers.writeJSON "contrast-node-install.json" {
          files = [
            {
              url = "file:///opt/edgeless/share/kata-containers.img";
              path = "/opt/edgeless/@@runtimeName@@/share/kata-containers.img";
            }
            {
              url = "file:///opt/edgeless/share/kata-containers-igvm.img";
              path = "/opt/edgeless/@@runtimeName@@/share/kata-containers-igvm.img";
            }
            {
              url = "file:///opt/edgeless/bin/cloud-hypervisor-snp";
              path = "/opt/edgeless/@@runtimeName@@/bin/cloud-hypervisor-snp";
              executable = true;
            }
            {
              url = "file:///opt/edgeless/bin/containerd-shim-contrast-cc-v2";
              path = "/opt/edgeless/@@runtimeName@@/bin/containerd-shim-contrast-cc-v2";
              executable = true;
            }
          ];
          inherit debugRuntime;
        };
        destination = "/config/contrast-node-install.json";
      }
    ];
  };

  kata-container-img = ociLayerTar {
    files = [
      {
        source = microsoft.kata-image;
        destination = "/opt/edgeless/share/kata-containers.img";
      }
      {
        source =
          if debugRuntime then (microsoft.kata-igvm.override { debug = true; }) else microsoft.kata-igvm;
        destination = "/opt/edgeless/share/kata-containers-igvm.img";
      }
    ];
  };

  cloud-hypervisor = ociLayerTar {
    files = [
      {
        source = lib.getExe microsoft.cloud-hypervisor;
        destination = "/opt/edgeless/bin/cloud-hypervisor-snp";
      }
    ];
  };

  containerd-shim = ociLayerTar {
    files = [
      {
        source = lib.getExe microsoft.kata-runtime;
        destination = "/opt/edgeless/bin/containerd-shim-contrast-cc-v2";
      }
    ];
  };

  version = ociLayerTar {
    files = [
      {
        source = ../../../../version.txt;
        destination = "/usr/share/misc/contrast/version.txt";
      }
    ];
  };

  layers = [
    installer-config
    kata-container-img
    cloud-hypervisor
    containerd-shim
    version
  ];

  manifest = ociImageManifest {
    layers = layers ++ [ node-installer ];
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
        "org.opencontainers.image.title" = "contrast-node-installer-microsoft";
        "org.opencontainers.image.description" = "Contrast Node Installer (Microsoft)";
      };
    };
  };
in

ociImageLayout {
  manifests = [ manifest ];
  passthru = {
    inherit debugRuntime;
    runtimeHash = hashDirs {
      dirs = layers; # Layers without node-installer, or we have a circular dependency!
      name = "runtime-hash-microsoft";
    };
  };
}
