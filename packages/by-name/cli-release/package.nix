# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  contrast,
  kata,
  microsoft,
}:

(contrast.overrideAttrs (
  _finalAttrs: _previousAttrs: {
    postPatch = ''
      install -D ${microsoft.genpolicy.settings}/genpolicy-settings.json cli/genpolicy/assets/genpolicy-settings-microsoft.json
      install -D ${kata.genpolicy.settings}/genpolicy-settings.json cli/genpolicy/assets/genpolicy-settings-kata.json
    '';
  }
)).cli
