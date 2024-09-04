# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ contrast }:

(contrast.overrideAttrs (
  _finalAttrs: previousAttrs: {
    postPatch = ''
      install -D ${contrast.settings}/genpolicy-settings.json cli/genpolicy/assets/genpolicy-settings.json
    '';

    ldflags = previousAttrs.ldflags ++ [
      "-X github.com/edgelesssys/contrast/cli/cmd.DefaultCoordinatorPolicyHash=${builtins.readFile ../../../cli/cmd/assets/coordinator-policy-hash}"
    ];
  }
)).cli
