# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

_final: prev: {
  contrastPkgs = prev.contrastPkgs.overrideScope (
    _final: prev: {
      kata = prev.kata.overrideScope (
        _final: kataPrev: {
          runtime = kataPrev.runtime-rs;
        }
      );
      contrast = prev.contrast.overrideScope (
        _final: prev: {
          nodeinstaller = prev.nodeinstaller.overrideAttrs (
            _finalAttrs: prevAttrs: {
              tags = prevAttrs.tags or [ ] ++ [ "runtimers" ];
            }
          );
        }
      );
    }
  );
}
