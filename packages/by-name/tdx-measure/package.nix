# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ buildGoModule }:

buildGoModule (finalAttrs: {
  pname = "tdx-measure";
  version = "0.1.0";

  src = ../../../tools/tdx-measure;

  proxyVendor = true;
  vendorHash = "sha256-Dt+M+zuEuDUs2hSRRewXeu/U4IIyMbROVxv1AV5tb44=";

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
