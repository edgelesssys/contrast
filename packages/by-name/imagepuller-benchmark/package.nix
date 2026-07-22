# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ lib, buildGoModule }:

buildGoModule (finalAttrs: {
  pname = "imagepuller-benchmark";
  version = "0.1.0";

  src =
    let
      inherit (lib) fileset path;
      root = ../../../.;
    in
    fileset.toSource {
      inherit root;
      fileset = fileset.unions [
        (path.append root "imagepuller/go.mod")
        (path.append root "imagepuller/go.sum")
        (path.append root "imagepuller/client")
        (path.append root "internal/katacomponents")
        (path.append root "go.mod")
        (path.append root "go.sum")
        (path.append root "tools/imagepuller-benchmark")
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-WZCGMEG1QZnFN94nbMEutY3AGueO2veBggrg3zlUw6w=";

  sourceRoot = "${finalAttrs.src.name}/tools/imagepuller-benchmark";

  env.CGO_ENABLED = 0;

  ldflags = [ "-s" ];

  doCheck = false;

  meta = lib.contrast.ourMeta { mainProgram = "imagepuller-benchmark"; };
})
