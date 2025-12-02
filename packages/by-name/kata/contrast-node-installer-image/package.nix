# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  ociLayerTar,
  ociImageManifest,
  ociImageLayout,
  writers,
  hashDirs,

  contrast,
  kata,
  pkgsStatic,
  contrastPkgsStatic,
  OVMF-SNP,
  OVMF-TDX,

  debugRuntime ? false,
  withGPU ? false,
}:

let
  os-image = kata.image.override {
    inherit withGPU;
    withDebug = debugRuntime;
  };

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
              url = "file:///opt/edgeless/share/kata-initrd.zst";
              path = "/opt/edgeless/@@runtimeName@@/share/kata-initrd.zst";
            }
            {
              url = "file:///opt/edgeless/snp/bin/qemu-system-x86_64";
              path = "/opt/edgeless/@@runtimeName@@/snp/bin/qemu-system-x86_64";
              executable = true;
            }
            {
              url = "file:///opt/edgeless/tdx/bin/qemu-system-x86_64";
              path = "/opt/edgeless/@@runtimeName@@/tdx/bin/qemu-system-x86_64";
              executable = true;
            }
            {
              url = "file:///opt/edgeless/snp/share/OVMF.fd";
              path = "/opt/edgeless/@@runtimeName@@/snp/share/OVMF.fd";
            }
            {
              url = "file:///opt/edgeless/tdx/share/OVMF.fd";
              path = "/opt/edgeless/@@runtimeName@@/tdx/share/OVMF.fd";
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
              url = "file:///opt/edgeless/snp/share/qemu/kvmvapic.bin";
              path = "/opt/edgeless/@@runtimeName@@/snp/share/qemu/kvmvapic.bin";
            }
            {
              url = "file:///opt/edgeless/snp/share/qemu/linuxboot_dma.bin";
              path = "/opt/edgeless/@@runtimeName@@/snp/share/qemu/linuxboot_dma.bin";
            }
            {
              url = "file:///opt/edgeless/snp/share/qemu/efi-virtio.rom";
              path = "/opt/edgeless/@@runtimeName@@/snp/share/qemu/efi-virtio.rom";
            }
            {
              url = "file:///opt/edgeless/tdx/share/qemu/kvmvapic.bin";
              path = "/opt/edgeless/@@runtimeName@@/tdx/share/qemu/kvmvapic.bin";
            }
            {
              url = "file:///opt/edgeless/tdx/share/qemu/linuxboot_dma.bin";
              path = "/opt/edgeless/@@runtimeName@@/tdx/share/qemu/linuxboot_dma.bin";
            }
            {
              url = "file:///opt/edgeless/tdx/share/qemu/efi-virtio.rom";
              path = "/opt/edgeless/@@runtimeName@@/tdx/share/qemu/efi-virtio.rom";
            }
          ];
          inherit debugRuntime;
          qemuExtraKernelParams = os-image.cmdline;
        };
        destination = "/config/contrast-node-install.json";
      }
    ];
  };

  kata-container-img = ociLayerTar {
    files = [
      {
        source = "${os-image.image}/${os-image.imageFileName}";
        destination = "/opt/edgeless/share/kata-containers.img";
      }
      {
        source = "${os-image.kernel}/bzImage";
        destination = "/opt/edgeless/share/kata-kernel";
      }
      {
        source = "${os-image.initialRamdisk}/initrd";
        destination = "/opt/edgeless/share/kata-initrd.zst";
      }
    ];
  };

  ovmf-snp = ociLayerTar {
    files = [
      {
        source = "${OVMF-SNP}/FV/OVMF.fd";
        destination = "/opt/edgeless/snp/share/OVMF.fd";
      }
    ];
  };

  qemu-snp = ociLayerTar {
    files = [
      {
        source = "${contrastPkgsStatic.qemu-snp}/bin/qemu-system-x86_64";
        destination = "/opt/edgeless/snp/bin/qemu-system-x86_64";
      }
      {
        source = "${contrastPkgsStatic.qemu-snp}/share/qemu/kvmvapic.bin";
        destination = "/opt/edgeless/snp/share/qemu/kvmvapic.bin";
      }
      {
        source = "${contrastPkgsStatic.qemu-snp}/share/qemu/linuxboot_dma.bin";
        destination = "/opt/edgeless/snp/share/qemu/linuxboot_dma.bin";
      }
      {
        source = "${contrastPkgsStatic.qemu-snp}/share/qemu/efi-virtio.rom";
        destination = "/opt/edgeless/snp/share/qemu/efi-virtio.rom";
      }
    ];
  };

  ovmf-tdx = ociLayerTar {
    files = [
      {
        source = "${OVMF-TDX}/FV/OVMF.fd";
        destination = "/opt/edgeless/tdx/share/OVMF.fd";
      }
    ];
  };

  qemu-tdx = ociLayerTar {
    files = [
      {
        source = "${contrastPkgsStatic.qemu-tdx}/bin/qemu-system-x86_64";
        destination = "/opt/edgeless/tdx/bin/qemu-system-x86_64";
      }
      {
        source = "${contrastPkgsStatic.qemu-tdx}/share/qemu/kvmvapic.bin";
        destination = "/opt/edgeless/tdx/share/qemu/kvmvapic.bin";
      }
      {
        source = "${contrastPkgsStatic.qemu-tdx}/share/qemu/linuxboot_dma.bin";
        destination = "/opt/edgeless/tdx/share/qemu/linuxboot_dma.bin";
      }
      {
        source = "${contrastPkgsStatic.qemu-tdx}/share/qemu/efi-virtio.rom";
        destination = "/opt/edgeless/tdx/share/qemu/efi-virtio.rom";
      }
    ];
  };

  kata-runtime = ociLayerTar {
    files = [
      {
        source = "${contrastPkgsStatic.kata.runtime}/bin/kata-runtime";
        destination = "/opt/edgeless/bin/kata-runtime";
      }
      {
        source = "${contrastPkgsStatic.kata.runtime}/bin/containerd-shim-kata-v2";
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
    ovmf-snp
    ovmf-tdx
    qemu-snp
    qemu-tdx
    kata-runtime
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
        "org.opencontainers.image.title" = "contrast-node-installer-kata";
        "org.opencontainers.image.description" = "Contrast Node Installer (Kata)";
      };
    };
  };
in

ociImageLayout {
  manifests = [ manifest ];
  passthru = {
    inherit debugRuntime os-image;
    runtimeHash = hashDirs {
      # Layers without node-installer, or we have a circular dependency!
      # To still account for node-installer changes in the runtime hash,
      # we include its src instead.
      dirs = layers ++ [ contrast.nodeinstaller.src ];
      name = "runtime-hash-kata";
    };
    gpu = kata.contrast-node-installer-image.override {
      inherit debugRuntime;
      withGPU = true;
    };
  };
}
