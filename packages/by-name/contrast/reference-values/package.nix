# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  kata,
  OVMF-TDX,
}:

let
  runtimeHandler =
    platform: hashFile:
    "contrast-cc-${platform}-${builtins.substring 0 8 (builtins.readFile hashFile)}";

  metal-qemu-tdx-handler = runtimeHandler "metal-qemu-tdx" kata.contrast-node-installer-image.runtimeHash;
  metal-qemu-snp-handler = runtimeHandler "metal-qemu-snp" kata.contrast-node-installer-image.runtimeHash;
  metal-qemu-snp-gpu-handler = runtimeHandler "metal-qemu-snp-gpu" kata.contrast-node-installer-image.runtimeHash;
  metal-qemu-tdx-gpu-handler = runtimeHandler "metal-qemu-tdx-gpu" kata.contrast-node-installer-image.runtimeHash;

  snpRefValsWith = os-image: {
    snp =
      let
        guestPolicy = builtins.fromJSON (builtins.readFile ./snpGuestPolicyQEMU.json);
        platformInfo = {
          SMTEnabled = true;
        };
        launch-digest = kata.calculateSnpLaunchDigest {
          inherit os-image;
          debug = kata.contrast-node-installer-image.debugRuntime;
        };
      in
      [
        {
          inherit guestPolicy platformInfo;
          trustedMeasurement = builtins.readFile "${launch-digest}/milan.hex";
          productName = "Milan";
        }
        {
          inherit guestPolicy platformInfo;
          trustedMeasurement = builtins.readFile "${launch-digest}/genoa.hex";
          productName = "Genoa";
        }
      ];
  };

  snpRefVals = snpRefValsWith kata.contrast-node-installer-image.os-image;
  snpGpuRefVals = snpRefValsWith kata.contrast-node-installer-image.gpu.os-image;

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
              debug = kata.contrast-node-installer-image.debugRuntime;
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
          }
        )
      ];
    };
  tdxRefVals = tdxRefValsWith {
    inherit (kata.contrast-node-installer-image) os-image;
    ovmf = OVMF-TDX;
    withGPU = false;
  };
  tdxGpuRefVals = tdxRefValsWith {
    inherit (kata.contrast-node-installer-image.gpu) os-image;
    ovmf = OVMF-TDX.override {
      # Only enable ACPI verification for the GPU build, until
      # the verification is actually secure.
      withACPIVerificationInsecure = true;
    };
    withGPU = true;
  };
in

builtins.toFile "reference-values.json" (
  builtins.toJSON {
    "${metal-qemu-tdx-handler}" = tdxRefVals;
    "${metal-qemu-snp-handler}" = snpRefVals;
    "${metal-qemu-snp-gpu-handler}" = snpGpuRefVals;
    "${metal-qemu-tdx-gpu-handler}" = tdxGpuRefVals;
  }
)
