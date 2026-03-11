# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

_final: prev: {
  contrastPkgs = prev.contrastPkgs.overrideScope (
    _final: prev: {
      kata =
        let
          kata-runtime = prev.kata.runtime;
          runtime-rs = prev.kata.runtime-rs.override {
            runtime = kata-runtime;
          };
        in
        prev.kata.overrideScope (
          _final: _prev: {
            runtime = runtime-rs;
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
