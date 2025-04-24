# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  mkShell,
  yarn,
}:

mkShell {
  packages = [ yarn ];
  shellHook = ''
    yarn install
  '';
}
