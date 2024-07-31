# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  contrast,
  microsoft,
  genpolicy ? microsoft.genpolicy,
}:

(contrast.overrideAttrs (
  _finalAttrs: previousAttrs: {
    prePatch = ''
      install -D ${lib.getExe genpolicy} cli/genpolicy/assets/genpolicy
      install -D ${contrast.settings}/genpolicy-settings.json cli/genpolicy/assets/genpolicy-settings.json
      install -D ${contrast.rules}/genpolicy-rules.rego cli/genpolicy/assets/genpolicy-rules.rego
      # TODO(burgerdev): cli/genpolicy/assets/allow-all.rego is insecure and deliberately omitted
      install -D ${contrast.embeddedReferenceValues} internal/manifest/assets/reference-values.json
    '';

    ldflags = previousAttrs.ldflags ++ [
      "-X github.com/edgelesssys/contrast/cli/cmd.DefaultCoordinatorPolicyHash=${builtins.readFile ../../../cli/cmd/assets/coordinator-policy-hash}"
    ];
  }
)).cli
