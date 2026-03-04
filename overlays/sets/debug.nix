# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

_final: prev: {
  contrastPkgs = prev.contrastPkgs.overrideScope (
    _final: prev: {
      contrast = prev.contrast.overrideScope (
        _final: prev: {
          node-installer-image = prev.node-installer-image.override {
            withDebug = true;
          };
        }
      );
    }
  );
}
