# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  kata,
  node-installer-image,
}:

let
  # Compute launch digests for all vCPU counts and CPU products for a given OS image.
  # The guest policy is no longer baked in here, instead being read from the manifest at generate time.
  snpLaunchDigestsFor =
    os-image:
    let
      digestsForVcpus =
        vcpus:
        let
          launch-digest = kata.calculateSnpLaunchDigest {
            inherit os-image vcpus;
            inherit (node-installer-image) withDebug;
          };
        in
        {
          Milan = {
            launchDigest = lib.strings.trim (builtins.readFile "${launch-digest}/milan.hex");
          };
          Genoa = {
            launchDigest = lib.strings.trim (builtins.readFile "${launch-digest}/genoa.hex");
          };
        };

      vcpuCounts = lib.range 1 8;
      allVcpuBlocks = builtins.listToAttrs (
        map (vcpus: {
          name = toString vcpus;
          value = digestsForVcpus vcpus;
        }) vcpuCounts
      );
    in
    allVcpuBlocks;
in

builtins.toFile "snp-launch-digests.json" (
  builtins.toJSON {
    metal-qemu-snp = snpLaunchDigestsFor node-installer-image.os-image;
    metal-qemu-snp-gpu = snpLaunchDigestsFor node-installer-image.gpu.os-image;
  }
)
