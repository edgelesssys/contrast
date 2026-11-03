# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  mkShell,
  nodejs,
  yarn-berry_4,
}:

mkShell {
  packages = [
    nodejs
    yarn-berry_4
  ];
  shellHook = ''
    yarn --cwd "$(git rev-parse --show-toplevel)/docs" install
  '';
}
