# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  ociLayerTar,
  ociImageManifest,
  ociImageLayout,
  writers,
  hashDirs,

  contrast,
  kata,
  nydus-snapshotter,
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
              url = "file:///opt/edgeless/share/kata-kernel";
              path = "/opt/edgeless/@@runtimeName@@/share/kata-kernel";
            }
            {
              url = "file:///opt/edgeless/bin/containerd-shim-contrast-cc-v2";
              path = "/opt/edgeless/@@runtimeName@@/bin/containerd-shim-contrast-cc-v2";
              executable = true;
            }
            {
              url = "file:///opt/edgeless/bin/kata-runtime";
              path = "/opt/edgeless/@@runtimeName@@/bin/kata-runtime";
              executable = true;
            }
            {
              url = "file:///bin/nydus-overlayfs";
              path = "/opt/edgeless/@@runtimeName@@/bin/nydus-overlayfs";
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
        source = kata.kata-image;
        destination = "/opt/edgeless/share/kata-containers.img";
      }
      {
        source = "${kata.kata-kernel-uvm}/bzImage";
        destination = "/opt/edgeless/share/kata-kernel";
      }
    ];
  };

  kata-runtime = ociLayerTar {
    files = [
      {
        source = "${kata.kata-runtime}/bin/kata-runtime";
        destination = "/opt/edgeless/bin/kata-runtime";
      }
      {
        source = "${kata.kata-runtime}/bin/containerd-shim-kata-v2";
        destination = "/opt/edgeless/bin/containerd-shim-contrast-cc-v2";
      }
    ];
  };

  nydus = ociLayerTar {
    files = [
      {
        source = "${nydus-snapshotter}/bin/nydus-overlayfs";
        destination = "/bin/nydus-overlayfs";
      }
    ];
  };

  layers = [
    installer-config
    kata-container-img
    kata-runtime
    nydus
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
        "org.opencontainers.image.title" = "contrast-node-installer-peerpod";
        "org.opencontainers.image.description" = "Contrast Node Installer (Peerpod)";
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
      name = "runtime-hash-peerpod";
    };
  };
}
