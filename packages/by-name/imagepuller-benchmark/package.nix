# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ lib, buildGoModule }:

buildGoModule (finalAttrs: {
  pname = "imagepuller-benchmark";
  version = "0.1.0";

  # The source of the main module of this repo. We filter for Go files so that
  # changes in the other parts of this repo don't trigger a rebuild.
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
        (path.append root "imagepuller/internal/api")
        (path.append root "tools/imagepuller-benchmark")
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-zTLSiMOsuXAdMRX8EWYnZTJzg4zAs6u5vao66gPADQw=";

  sourceRoot = "${finalAttrs.src.name}/tools/imagepuller-benchmark";

  env.CGO_ENABLED = 0;

  ldflags = [ "-s" ];

  doCheck = false;

  meta.mainProgram = "imagepuller-benchmark";
})
