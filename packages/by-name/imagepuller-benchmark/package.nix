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
        (path.append root "imagepuller/internal/imagepullapi")
        (path.append root "tools/imagepuller-benchmark")
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-auGTTfEFnIewCj+JHclI2mZCO7jpUqWIVnUc2+L3QOI=";

  sourceRoot = "${finalAttrs.src.name}/tools/imagepuller-benchmark";

  env.CGO_ENABLED = 0;

  ldflags = [ "-s" ];

  doCheck = false;

  meta.mainProgram = "imagepuller-benchmark";
})
