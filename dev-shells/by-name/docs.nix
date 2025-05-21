# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

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
