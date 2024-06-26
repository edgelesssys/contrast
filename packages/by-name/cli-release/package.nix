# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ lib
, contrast
, microsoft
, genpolicy ? microsoft.genpolicy
}:

(contrast.overrideAttrs (_finalAttrs: previousAttrs: {
  prePatch = ''
    install -D ${lib.getExe genpolicy} cli/cmd/assets/genpolicy
    install -D ${contrast.settings}/genpolicy-settings.json cli/cmd/assets/genpolicy-settings.json
    install -D ${contrast.rules}/genpolicy-rules.rego cli/cmd/assets/genpolicy-rules.rego
  '';

  ldflags = previousAttrs.ldflags ++ [
    "-X github.com/edgelesssys/contrast/cli/cmd.DefaultCoordinatorPolicyHash=${builtins.readFile ../../../cli/cmd/assets/coordinator-policy-hash}"
  ];
})).cli
