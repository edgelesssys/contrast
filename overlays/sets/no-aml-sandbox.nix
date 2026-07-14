# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

# no-aml-sandbox disables the guest's AML sandbox. It is meant to be combined
# with the badaml set (i.e. `badaml+no-aml-sandbox`) to make the guest
# vulnerable to the BadAML attack.

_final: prev: {
  contrastPkgs = prev.contrastPkgs.overrideScope (
    _contrastPkgsFinal: contrastPkgsPrev: {
      kata = contrastPkgsPrev.kata.overrideScope (
        _kataFinal: kataPrev: {
          kernel-uvm = kataPrev.kernel-uvm.override {
            withAMLSandbox = false;
          };
        }
      );
    }
  );
}
