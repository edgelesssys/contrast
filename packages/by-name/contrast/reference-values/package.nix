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
  snpRefValsWith =
    os-image:
    let
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

    in
    runCommand "snp-reference-values.json" { nativeBuildInputs = [ jq ]; } ''
      set -o pipefail
      {
      ${lib.concatMapStringsSep "\n" (product: ''
        jq -n \
          --slurpfile guestPolicy ${./snpGuestPolicyQEMU.json} \
          --arg productName ${lib.escapeShellArg product} \
          --rawfile measurement ${launch-digest-1vcpu}/${lib.toLower product}.hex \
          '{guestPolicy: $guestPolicy[0], platformInfo: {SMTEnabled: true}, productName: $productName, TrustedMeasurement: $measurement}'
      '') products}
      } | jq -s '{snp: .}' > "$out"
    '';

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
      in
      runCommand "tdx-reference-values.json" { nativeBuildInputs = [ jq ]; } ''
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
      ''
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
in

runCommand "reference-values.json" { nativeBuildInputs = [ jq ]; } ''
  jq -n \
    --rawfile runtimeHash ${node-installer-image.runtimeHash} \
    --slurpfile tdx ${tdxRefVals} \
    --slurpfile snp ${snpRefVals} \
    --slurpfile snpGpu ${snpGpuRefVals} \
    --slurpfile tdxGpu ${tdxGpuRefVals} \
    '
      def handler(platform): "contrast-" + platform + "-" + $runtimeHash[0:8];
      {
        (handler("cc-metal-qemu-tdx")): $tdx[0],
        (handler("cc-metal-qemu-snp")): $snp[0],
        (handler("cc-metal-qemu-snp-gpu")): $snpGpu[0],
        (handler("cc-metal-qemu-tdx-gpu")): $tdxGpu[0],
        (handler("insecure-metal-qemu")): {snp: [{}]},
        (handler("insecure-metal-qemu-gpu")): {snp: [{}]}
      }
    ' > "$out"
''
