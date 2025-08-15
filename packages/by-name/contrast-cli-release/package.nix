# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  contrast,
  kata,
}:

(contrast.overrideAttrs (
  _finalAttrs: _previousAttrs: {
    postPatch = ''
      install -D ${kata.genpolicy.settings}/genpolicy-settings.json cli/genpolicy/assets/genpolicy-settings-kata.json
    '';
  }
)).cli
