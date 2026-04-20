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
  insecure-metal-qemu-snp-handler = runtimeHandler "insecure-metal-qemu-snp" node-installer-image.runtimeHash;
  insecure-metal-qemu-snp-gpu-handler = runtimeHandler "insecure-metal-qemu-snp-gpu" node-installer-image.runtimeHash;
  insecure-metal-qemu-tdx-handler = runtimeHandler "insecure-metal-qemu-tdx" node-installer-image.runtimeHash;
  insecure-metal-qemu-tdx-gpu-handler = runtimeHandler "insecure-metal-qemu-tdx-gpu" node-installer-image.runtimeHash;

  snpRefValsWith = os-image: {
    snp =
      let
        guestPolicy = builtins.fromJSON (builtins.readFile ./snpGuestPolicyQEMU.json);
        platformInfo = {
          SMTEnabled = true;
        };
        vcpuCounts = lib.range 1 8;
        products = [
          "Milan"
          "Genoa"
        ];

        generateRefVal =
          vcpus: product:
          let
            launch-digest = kata.calculateSnpLaunchDigest {
              inherit os-image vcpus;
              inherit (node-installer-image) withDebug;
            };
            filename = "${lib.toLower product}.hex";
          in
          {
            inherit guestPolicy platformInfo;
            trustedMeasurement = builtins.readFile "${launch-digest}/${filename}";
            productName = product;
            cpus = vcpus;
          };
      in
      builtins.concatLists (map (vcpus: map (product: generateRefVal vcpus product) products) vcpuCounts);
  };

  snpRefVals = snpRefValsWith node-installer-image.os-image;
  snpGpuRefVals = snpRefValsWith node-installer-image.gpu.os-image;

  tdxRefValsWith =
    {
      os-image,
      ovmf,
      withGPU,
    }:
    {
      tdx = [
        (
          let
            launch-digests = kata.calculateTdxLaunchDigests {
              inherit os-image ovmf withGPU;
              inherit (node-installer-image) withDebug;
            };
          in
          {
            mrTd = builtins.readFile "${launch-digests}/mrtd.hex";
            rtmrs = [
              (builtins.readFile "${launch-digests}/rtmr0.hex")
              (builtins.readFile "${launch-digests}/rtmr1.hex")
              (builtins.readFile "${launch-digests}/rtmr2.hex")
              (builtins.readFile "${launch-digests}/rtmr3.hex")
            ];
            xfam = "e702060000000000";
            memoryIntegrity = false;
          }
        )
      ];
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
  insecureSnpRefVals = {
    snp = [ { } ];
  };
  insecureTdxRefVals = {
    tdx = [ { } ];
  };
in

builtins.toFile "reference-values.json" (
  builtins.toJSON {
    "${cc-metal-qemu-tdx-handler}" = tdxRefVals;
    "${cc-metal-qemu-snp-handler}" = snpRefVals;
    "${cc-metal-qemu-snp-gpu-handler}" = snpGpuRefVals;
    "${cc-metal-qemu-tdx-gpu-handler}" = tdxGpuRefVals;
    "${insecure-metal-qemu-snp-handler}" = insecureSnpRefVals;
    "${insecure-metal-qemu-snp-gpu-handler}" = insecureSnpRefVals;
    "${insecure-metal-qemu-tdx-handler}" = insecureTdxRefVals;
    "${insecure-metal-qemu-tdx-gpu-handler}" = insecureTdxRefVals;
  }
)
