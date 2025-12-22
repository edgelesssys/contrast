# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ inputs }:

final: _prev:

{
  contrastPkgs = import ../packages { pkgs = final; };
  lib = import ../lib { inherit inputs; };
}
