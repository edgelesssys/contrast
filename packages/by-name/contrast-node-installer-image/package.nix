# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ lib
, ociLayerTar
, ociImageManifest
, ociImageLayout
, contrast-node-installer
, runtime-class-files
, pkgsStatic
, writers
}:

let
  node-installer = ociLayerTar {
    files = [
      { source = lib.getExe contrast-node-installer; destination = "/bin/node-installer"; }
      { source = "${pkgsStatic.util-linux}/bin/nsenter"; destination = "/bin/nsenter"; }
    ];
  };

  launch-digest = lib.removeSuffix "\n" (builtins.readFile "${runtime-class-files}/launch-digest.hex");
  runtime-handler = lib.removeSuffix "\n" (builtins.readFile "${runtime-class-files}/runtime-handler");

  installer-config = ociLayerTar {
    files = [
      {
        source = writers.writeJSON "contrast-node-install.json" {
          files = [
            { url = "file:///opt/edgeless/share/kata-containers.img"; path = "/opt/edgeless/${runtime-handler}/share/kata-containers.img"; }
            { url = "file:///opt/edgeless/share/kata-containers-igvm.img"; path = "/opt/edgeless/${runtime-handler}/share/kata-containers-igvm.img"; }
            { url = "file:///opt/edgeless/bin/cloud-hypervisor-snp"; path = "/opt/edgeless/${runtime-handler}/bin/cloud-hypervisor-snp"; }
            { url = "file:///opt/edgeless/bin/containerd-shim-contrast-cc-v2"; path = "/opt/edgeless/${runtime-handler}/bin/containerd-shim-contrast-cc-v2"; }
          ];
          runtimeHandlerName = runtime-handler;
          inherit (runtime-class-files) debugRuntime;
        };
        destination = "/config/contrast-node-install.json";
      }
    ];
  };

  kata-container-img = ociLayerTar {
    files = [
      { source = runtime-class-files.rootfs; destination = "/opt/edgeless/share/kata-containers.img"; }
      { source = runtime-class-files.igvm; destination = "/opt/edgeless/share/kata-containers-igvm.img"; }
    ];
  };

  cloud-hypervisor = ociLayerTar {
    files = [
      { source = runtime-class-files.cloud-hypervisor-exe; destination = "/opt/edgeless/bin/cloud-hypervisor-snp"; }
    ];
  };

  containerd-shim = ociLayerTar {
    files = [{ source = runtime-class-files.containerd-shim-contrast-cc-v2; destination = "/opt/edgeless/bin/containerd-shim-contrast-cc-v2"; }];
  };

  manifest = ociImageManifest
    {
      layers = [
        node-installer
        installer-config
        kata-container-img
        cloud-hypervisor
        containerd-shim
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
          "org.opencontainers.image.title" = "contrast-node-installer";
          "org.opencontainers.image.description" = "Contrast Node Installer";
          "systems.edgeless.contrast.snp-launch-digest" = launch-digest;
        };
      };
    };
in

ociImageLayout {
  manifests = [ manifest ];
}
