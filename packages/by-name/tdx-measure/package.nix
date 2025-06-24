# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ buildGoModule }:

buildGoModule (finalAttrs: {
  pname = "tdx-measure";
  version = "0.1.0";

  src = ../../../tools/tdx-measure;

  proxyVendor = true;
  vendorHash = "sha256-nUNr2vfhCWtpUvIDppONh/x5v1JODuLFwc9tjZK05Vg=";

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
