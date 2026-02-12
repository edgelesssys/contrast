# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  kata,
  calculateSnpIDBlock,
  node-installer-image,
}:

let
  snpIdBlocksFor =
    os-image:
    let
      guestPolicy = builtins.fromJSON (builtins.readFile ../reference-values/snpGuestPolicyQEMU.json);
      idBlocksForVcpus =
        vcpus:
        let
          launch-digest = kata.calculateSnpLaunchDigest {
            inherit os-image vcpus;
            debug = node-installer-image.debugRuntime;
          };
          idBlocks = calculateSnpIDBlock {
            snp-launch-digest = launch-digest;
            snp-guest-policy = ../reference-values/snpGuestPolicyQEMU.json;
          };
        in
        {
          Milan = {
            idBlock = builtins.readFile "${idBlocks}/id-block-milan.base64";
            idAuth = builtins.readFile "${idBlocks}/id-auth-milan.base64";
            inherit guestPolicy;
          };
          Genoa = {
            idBlock = builtins.readFile "${idBlocks}/id-block-genoa.base64";
            idAuth = builtins.readFile "${idBlocks}/id-auth-genoa.base64";
            inherit guestPolicy;
          };
        };

      vcpuCounts = builtins.genList (x: x + 1) 8;
      allVcpuBlocks = builtins.listToAttrs (
        map (vcpus: {
          name = toString vcpus;
          value = idBlocksForVcpus vcpus;
        }) vcpuCounts
      );
    in
    allVcpuBlocks;
in

builtins.toFile "snp-id-blocks.json" (
  builtins.toJSON {
    metal-qemu-snp = snpIdBlocksFor node-installer-image.os-image;
    metal-qemu-snp-gpu = snpIdBlocksFor node-installer-image.gpu.os-image;
  }
)
