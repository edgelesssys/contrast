# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  contrast,
  kata,
  microsoft,
}:

let
  coordinatorPolicyHashesFallback = builtins.toFile "coordinator-policy-hashes-fallback.json" (
    builtins.toJSON {
      AKS-CLH-SNP = lib.trim (builtins.readFile ../../../cli/cmd/assets/coordinator-policy-hash);
      K3s-QEMU-TDX = "";
      K3s-QEMU-SNP = "";
      K3s-QEMU-SNP-GPU = "";
      RKE2-QEMU-TDX = "";
      Metal-QEMU-SNP = "";
      Metal-QEMU-SNP-GPU = "";
      Metal-QEMU-TDX = "";
    }
  );
in

(contrast.overrideAttrs (
  _finalAttrs: _previousAttrs: {
    postPatch = ''
      install -D ${microsoft.genpolicy.settings}/genpolicy-settings.json cli/genpolicy/assets/genpolicy-settings-microsoft.json
      install -D ${kata.genpolicy.settings}/genpolicy-settings.json cli/genpolicy/assets/genpolicy-settings-kata.json
      install -D ${coordinatorPolicyHashesFallback} cli/cmd/assets/coordinator-policy-hashes-fallback.json
    '';
  }
)).cli
