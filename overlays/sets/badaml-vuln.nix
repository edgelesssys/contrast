# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

let
  badamlScope = import ./_badaml.nix;
in
_final: prev: {
  # Deactivate the AML sandbox so the guest is vulnerable to BadAML attack.
  contrastPkgs = prev.contrastPkgs.overrideScope (badamlScope {
    withAMLSandbox = false;
  });
}
