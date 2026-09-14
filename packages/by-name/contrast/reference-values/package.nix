# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  jq,
  kata,
  OVMF-TDX,
  node-installer-image,
  runCommand,
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
    (
      let
        vcpuCounts = if withGPU then [ 1 ] else lib.range 1 220;
        launchDigests = map (
          vcpus:
          kata.calculateTdxLaunchDigests {
            inherit
              os-image
              ovmf
              withGPU
              vcpus
              ;
            inherit (node-installer-image) withDebug;
            rtmr0Only = vcpus != 1;
          }
        ) vcpuCounts;
        referenceValues = runCommand "tdx-reference-values.json" { nativeBuildInputs = [ jq ]; } ''
          set -o pipefail
          sharedLaunchDigests=${builtins.head launchDigests}
          for launchDigests in ${lib.escapeShellArgs (map toString launchDigests)}; do
            jq -n \
              --rawfile mrTd "$sharedLaunchDigests/mrtd.hex" \
              --rawfile rtmr0 "$launchDigests/rtmr0.hex" \
              --rawfile rtmr1 "$sharedLaunchDigests/rtmr1.hex" \
              --rawfile rtmr2 "$sharedLaunchDigests/rtmr2.hex" \
              --rawfile rtmr3 "$sharedLaunchDigests/rtmr3.hex" \
              '{
                mrTd: $mrTd,
                rtmrs: [$rtmr0, $rtmr1, $rtmr2, $rtmr3],
                # CET (XFAM bits 11/12) is ignored during validation because it depends on host CPU support. see validateXfamIgnoringCET.
                xfam: "e702060000000000",
                memoryIntegrity: false
              }'
          done | jq -s '
            length as $count
            | if (map(.rtmrs[0]) | unique | length) != $count then
              error("duplicate RTMR0 values")
            else
              .[0] as $shared
              | {
                  tdx: [
                    $shared + {
                      rtmr0Alternatives: (map(.rtmrs[0]) | .[1:])
                    }
                  ]
                }
            end
          ' > "$out"
        '';
      in
      builtins.fromJSON (builtins.readFile referenceValues)
    );
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
