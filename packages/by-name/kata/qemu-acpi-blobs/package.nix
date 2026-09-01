# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  buildGoModule,
  lib,
  stdenvNoCC,
  qemu-cc,
  source,
  tdx-measure,
}:

{
  # MADT depends on the vCPU count.
  vcpus,
  legacySerial ? false,
  # qemu-cc omits memory-dependent ACPI data; 130 MiB is the DMA minimum.
  memoryMiB ? 1024,
}:

# Generates OVMF-measured ACPI blobs for Kata's q35 topology with qtest.
let
  # Force review of the mirrored topology on Kata upgrades.
  mirroredKataVersion = "4.0.0";

  qemuAcpiDump = buildGoModule {
    pname = "qemu-acpi-dump";
    version = "0.1.0";

    src = ../../../../tools/tdx-measure;

    proxyVendor = true;
    vendorHash = "sha256-FNOA2wkqCk65e2qV+mezlmv2NZ96/K/fV8ocpVDwpAs=";

    subPackages = [ "internal/cmd/qemu-acpi-dump" ];

    env.CGO_ENABLED = 0;
    ldflags = [ "-s" ];
    doCheck = false;
  };

  mkBlob =
    {
      vcpus,
      legacySerial,
      memoryMiB,
    }:
    let
      mem = "${toString memoryMiB}M";

      # Legacy serial replaces virtio-serial-pci and shifts later PCI slots.
      serialDevices =
        if legacySerial then
          [
            "-chardev"
            "null,id=charconsole0"
            "-serial"
            "chardev:charconsole0"
          ]
        else
          [
            "-device"
            "virtio-serial-pci,disable-modern=false,id=serial0"
          ];

      # Device order mirrors Kata and determines ACPI PCI slots.
      qemuArgs = [
        # TDX disables SMM and PIC; both affect ACPI.
        "-machine"
        "q35,accel=qtest,smm=off,pic=off"
        # qtest cannot use -cpu host; ACPI uses the -smp count.
        "-cpu"
        "qemu64"
        "-m"
        mem
        # Match Kata's OVMF hot-plug bridge.
        "-device"
        "pcie-root-port,id=rp-pci-bridge-0,bus=pcie.0,chassis=16,slot=0,addr=2,multifunction=off,bus-reserve=0x1,pref64-reserve=1m,mem-reserve=1m,io-reserve=4k"
        "-device"
        "pcie-pci-bridge,bus=rp-pci-bridge-0,id=pci-bridge-0"
      ]
      ++ serialDevices
      ++ [
        # Storage devices preserve Kata's PCI topology; null-co provides their backends.
        "-blockdev"
        "driver=null-co,node-name=drv-image,size=1073741824"
        "-device"
        "virtio-blk-pci,disable-modern=false,drive=drv-image"
        "-device"
        "virtio-scsi-pci,id=scsi0,disable-modern=false"
        "-blockdev"
        "driver=null-co,node-name=drv-initdata,size=1073741824"
        "-device"
        "virtio-blk-pci,disable-modern=false,drive=drv-initdata"
        "-blockdev"
        "driver=null-co,node-name=drv-imagepuller,size=1073741824"
        "-device"
        "virtio-blk-pci,disable-modern=false,drive=drv-imagepuller"
        # Balloon preserves vhost-vsock's PCI slot without a host backend.
        "-device"
        "virtio-balloon-pci"
        # hubport replaces tap; virtio-net-pci remains ACPI-visible.
        "-netdev"
        "hubport,id=network-0,hubid=0"
        "-device"
        "virtio-net-pci,netdev=network-0,disable-modern=false"
        "-rtc"
        "base=utc,driftfix=slew,clock=host"
        "-vga"
        "none"
        "-no-user-config"
        "-nodefaults"
        "-nographic"
        "--no-reboot"
        "-object"
        "memory-backend-ram,id=dimm1,size=${mem}"
        "-numa"
        "node,memdev=dimm1"
        "-smp"
        "${toString vcpus},cores=${toString vcpus},threads=1,sockets=1,maxcpus=${toString vcpus}"
      ];

      manifest = builtins.toJSON {
        inherit vcpus legacySerial memoryMiB;
        kataVersion = source.version;
      };
    in
    assert lib.assertMsg (source.version == mirroredKataVersion)
      "qemu-acpi-blobs mirrors Kata ${mirroredKataVersion}; review qemuArgs before updating to ${source.version}";
    assert lib.assertMsg (vcpus > 0) "qemu-acpi-blobs requires vcpus > 0";
    assert lib.assertMsg (
      memoryMiB >= 130
    ) "qemu-acpi-blobs requires memoryMiB >= 130 for fw_cfg DMA scratch space";
    stdenvNoCC.mkDerivation {
      name = "qemu-acpi-blobs${lib.optionalString legacySerial "-legacy-serial"}-${toString vcpus}vcpu";

      dontUnpack = true;

      nativeBuildInputs = [
        qemu-cc
        qemuAcpiDump
      ];

      buildPhase = ''
        runHook preBuild

        mkdir -p "$out"
        qemu-acpi-dump \
          --output "$out" \
          --qemu ${qemu-cc}/bin/qemu-system-x86_64 \
          --metadata-json ${lib.escapeShellArg manifest} \
          -- ${lib.escapeShellArgs qemuArgs}

        runHook postBuild
      '';

      dontInstall = true;
    };
in
(mkBlob { inherit vcpus legacySerial memoryMiB; }).overrideAttrs (_old: {
  passthru.tests.regression = tdx-measure.overrideAttrs (_oldTdxMeasure: {
    pname = "qemu-acpi-blobs-test";
    # Reference digests use one vCPU; memory does not affect qemu-cc ACPI.
    ACPI_BLOBS_DEFAULT_DIR = mkBlob {
      vcpus = 1;
      legacySerial = false;
      inherit memoryMiB;
    };
    ACPI_BLOBS_LEGACY_SERIAL_DIR = mkBlob {
      vcpus = 1;
      legacySerial = true;
      inherit memoryMiB;
    };
    doCheck = true;
  });
})
