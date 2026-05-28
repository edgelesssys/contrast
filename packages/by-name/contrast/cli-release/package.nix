# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  cli,
  kata,
}:

(cli.overrideAttrs (_previousAttrs: {
  preConfigure = ''
    install -D ${kata.genpolicy.settings}/genpolicy-settings.json "''${modRoot:-.}/cli/genpolicy/assets/genpolicy-settings-kata.json"
  '';
}))
