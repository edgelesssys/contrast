# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  kata,
  OVMF-TDX,
  node-installer-image,
}:

let
  runtimeHandler =
    platform: hashFile: "contrast-${platform}-${builtins.substring 0 8 (builtins.readFile hashFile)}";

  cc-metal-qemu-tdx-handler = runtimeHandler "cc-metal-qemu-tdx" node-installer-image.runtimeHash;
  cc-metal-qemu-snp-handler = runtimeHandler "cc-metal-qemu-snp" node-installer-image.runtimeHash;
  cc-metal-qemu-snp-gpu-handler = runtimeHandler "cc-metal-qemu-snp-gpu" node-installer-image.runtimeHash;
  cc-metal-qemu-tdx-gpu-handler = runtimeHandler "cc-metal-qemu-tdx-gpu" node-installer-image.runtimeHash;
  insecure-metal-qemu-handler = runtimeHandler "insecure-metal-qemu" node-installer-image.runtimeHash;
  insecure-metal-qemu-gpu-handler = runtimeHandler "insecure-metal-qemu-gpu" node-installer-image.runtimeHash;

  snpRefValsWith = os-image: {
    snp =
      let
        guestPolicy = builtins.fromJSON (builtins.readFile ./snpGuestPolicyQEMU.json);
        platformInfo = {
          SMTEnabled = true;
        };
        products = [
          "Milan"
          "Genoa"
        ];

        # Compute the 1-vCPU launch digest once; all per-vCPU-count measurements
        # are derived at verify time using ExtendSNPLaunchDigest + APEIP.
        launch-digest-1vcpu = kata.calculateSnpLaunchDigest {
          inherit os-image;
          vcpus = 1;
          inherit (node-installer-image) withDebug;
        };

        generateRefVal =
          product:
          let
            filename = "${lib.toLower product}.hex";
          in
          {
            inherit guestPolicy platformInfo;
            productName = product;
            TrustedMeasurement = builtins.readFile "${launch-digest-1vcpu}/${filename}";
          };
      in
      map generateRefVal products;
  };

  snpRefVals = snpRefValsWith node-installer-image.os-image;
  snpGpuRefVals = snpRefValsWith node-installer-image.gpu.os-image;

  tdxRefValsWith =
    {
      os-image,
      ovmf,
      withGPU,
    }:
    let
      launch-digests = kata.calculateTdxLaunchDigests {
        inherit os-image ovmf withGPU;
        inherit (node-installer-image) withDebug;
      };
      mrTd = builtins.readFile "${launch-digests}/mrtd.hex";
      rtmr1 = builtins.readFile "${launch-digests}/rtmr1.hex";
      rtmr2 = builtins.readFile "${launch-digests}/rtmr2.hex";
      rtmr3 = builtins.readFile "${launch-digests}/rtmr3.hex";
      # RTMR[0] is emitted as a newline-separated list of candidates, one per possible extra-pci-roots / pxb-pcie count.
      rtmr0List = builtins.filter (s: s != "") (
        lib.splitString "\n" (builtins.readFile "${launch-digests}/rtmr0.hex")
      );
    in
    {
      tdx = map (rtmr0: {
        inherit mrTd;
        rtmrs = [
          rtmr0
          rtmr1
          rtmr2
          rtmr3
        ];
        # CET (XFAM bits 11/12) is ignored during validation because it depends on host CPU support. see validateXfamIgnoringCET.
        xfam = "e702060000000000";
        memoryIntegrity = false;
      }) rtmr0List;
    };
  tdxRefVals = tdxRefValsWith {
    inherit (node-installer-image) os-image;
    ovmf = OVMF-TDX;
    withGPU = false;
  };
  tdxGpuRefVals = tdxRefValsWith {
    inherit (node-installer-image.gpu) os-image;
    ovmf = OVMF-TDX.override {
      # Only enable ACPI verification for the GPU build, until
      # the verification is actually secure.
      withACPIVerificationInsecure = true;
    };
    withGPU = true;
  };
  insecureRefVals = {
    snp = [ { } ];
  };
in

builtins.toFile "reference-values.json" (
  builtins.toJSON {
    "${cc-metal-qemu-tdx-handler}" = tdxRefVals;
    "${cc-metal-qemu-snp-handler}" = snpRefVals;
    "${cc-metal-qemu-snp-gpu-handler}" = snpGpuRefVals;
    "${cc-metal-qemu-tdx-gpu-handler}" = tdxGpuRefVals;
    "${insecure-metal-qemu-handler}" = insecureRefVals;
    "${insecure-metal-qemu-gpu-handler}" = insecureRefVals;
  }
)
