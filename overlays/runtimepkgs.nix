# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

final: prev:

if prev.stdenv.hostPlatform.system == "x86_64-linux" then
  { }
else
  {
    contrastPkgs = prev.contrastPkgs.overrideScope (
      _cFinal: cPrev: {
        kata = cPrev.kata.overrideScope (
          _: _: {
            inherit (final.runtimePkgs.kata)
              contrast-node-installer-image
              agent
              image
              kernel-uvm
              calculateSnpLaunchDigest
              calculateTdxLaunchDigests
              ;
          }
        );

        contrast = cPrev.contrast.overrideScope (
          _: _: {
            inherit (final.runtimePkgs.contrast)
              coordinator
              initializer
              node-installer-image
              nodeinstaller
              ;
          }
        );

        inherit (final.runtimePkgs)
          debugshell
          service-mesh
          k8s-log-collector
          boot-image
          boot-microvm
          qemu-cc
          pause-bundle
          OVMF-TDX
          calculateSnpIDBlock
          ;

        scripts = cPrev.scripts.overrideScope (
          _: _: {
            inherit (final.runtimePkgs.scripts)
              cleanup-bare-metal
              cleanup-images
              cleanup-containerd
              nix-gc
              ;
          }
        );

        inherit (final.runtimePkgs) containers;
      }
    );
  }
