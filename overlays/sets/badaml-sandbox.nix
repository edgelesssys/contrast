# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

let
  badamlScope = import ./_badaml.nix;
in
_final: prev: {
  contrastPkgs = prev.contrastPkgs.overrideScope (badamlScope {
    withAMLSandbox = true;
  });
}
