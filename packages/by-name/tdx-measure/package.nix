# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ buildGoModule }:

buildGoModule (finalAttrs: {
  pname = "tdx-measure";
  version = "0.1.0";

  src = ../../../tools/tdx-measure;

  proxyVendor = true;
  vendorHash = "sha256-Z5Au1Uye/yIm8N52LAvKe2EJ4PbJ382afaJxAV9C1SM=";

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

  meta.mainProgram = "tdx-measure";
})
