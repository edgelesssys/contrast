# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  buildGoModule,
}:

buildGoModule (finalAttrs: {
  pname = "tdx-measure";
  version = "0.1.0";

  src = ../../../tools/tdx-measure;

  proxyVendor = true;
  vendorHash = "sha256-M2fqMG7oos/v1TzsElkrwnTK9/7ZJyWzwXkvpq9esYo=";

  subPackages = [ "." ];

  env.CGO_ENABLED = 0;

  ldflags = [
    "-s"
    "-X main.version=v${finalAttrs.version}"
  ];

  preCheck = ''
    export CGO_ENABLED=1
  '';

  checkPhase = ''
    runHook preCheck
    go test -race ./...
    runHook postCheck
  '';

  meta = lib.contrast.ourMeta { mainProgram = "tdx-measure"; };
})
