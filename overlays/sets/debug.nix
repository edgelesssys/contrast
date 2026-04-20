# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

_final: prev: {
  contrastPkgs = prev.contrastPkgs.overrideScope (
    _contrastPkgsFinal: contrastPkgsPrev: {
      # Build OVMF with debug output to serial port.
      OVMF-SNP = contrastPkgsPrev.OVMF-SNP.override {
        debug = true;
      };
      OVMF-TDX = contrastPkgsPrev.OVMF-TDX.override {
        debug = true;
      };
      contrast = contrastPkgsPrev.contrast.overrideScope (
        _contrastFinal: contrastPrev: {
          node-installer-image = contrastPrev.node-installer-image.override {
            withDebug = true;
          };
        }
      );
    }
  );
}
