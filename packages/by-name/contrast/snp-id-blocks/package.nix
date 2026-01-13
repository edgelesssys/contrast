# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  kata,
  calculateSnpIDBlock,
}:

let
  snpIdBlocksFor =
    os-image:
    let
      guestPolicy = builtins.fromJSON (builtins.readFile ../reference-values/snpGuestPolicyQEMU.json);
      launch-digest = kata.calculateSnpLaunchDigest {
        inherit os-image;
        debug = kata.contrast-node-installer-image.debugRuntime;
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
in

builtins.toFile "snp-id-blocks.json" (
  builtins.toJSON {
    metal-qemu-snp = snpIdBlocksFor kata.contrast-node-installer-image.os-image;
    metal-qemu-snp-gpu = snpIdBlocksFor kata.contrast-node-installer-image.gpu.os-image;
  }
)
