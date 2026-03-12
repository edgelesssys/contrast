# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ lib, buildGoModule }:

buildGoModule (finalAttrs: {
  pname = "kernelconfig";
  version = "0.1.0";

  src =
    let
      inherit (lib) fileset path;
      root = ../../../.;
    in
    fileset.toSource {
      inherit root;
      fileset = fileset.unions [
        (path.append root "tools/kernelconfig")
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-5BqvuvCMxJEgh6bYft+54oiMKxjXhxfcYfFaBZc+SMo=";

  sourceRoot = "${finalAttrs.src.name}/tools/kernelconfig";

  env.CGO_ENABLED = 0;

  ldflags = [ "-s" ];

  meta.mainProgram = "kernelconfig";
})
