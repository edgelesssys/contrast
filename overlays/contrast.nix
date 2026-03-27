# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

final: prev:

let
  # runtimePkgs is injected by the flake overlay and provides
  # the x86_64-linux set (via reverseContrastNesting), including
  # set-specific overrides. On x86_64-linux it falls back to final.
  runtimePkgs = prev.runtimePkgs or final;

  baseContrastPkgs = import ../packages { pkgs = final; };
in
if prev.stdenv.hostPlatform.system == "x86_64-linux" then
  { contrastPkgs = baseContrastPkgs; }
else
  {
    contrastPkgs = baseContrastPkgs.overrideScope (
      _cFinal: cPrev: {
        kata = cPrev.kata.overrideScope (
          _: _: {
            inherit (runtimePkgs.kata)
              contrast-node-installer-image
              agent
              image
              kernel-uvm
              ;
          }
        );

        contrast = cPrev.contrast.overrideScope (
          _: _: {
            inherit (runtimePkgs.contrast)
              coordinator
              docs
              initializer
              node-installer-image
              nodeinstaller
              reference-values
              snp-id-blocks
              ;
          }
        );

        inherit (runtimePkgs)
          debugshell
          tdx-tools
          service-mesh
          k8s-log-collector
          boot-image
          boot-microvm
          qemu-cc
          pause-bundle
          ;
      }
    );
  }
