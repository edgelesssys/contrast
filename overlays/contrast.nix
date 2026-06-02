# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

final: prev:

{
  contrastPkgs = import ../packages { pkgs = final; };
  lib = prev.lib.extend (
    finalLib: _: {
      contrast = import ../lib { lib = finalLib; };
    }
  );
}
