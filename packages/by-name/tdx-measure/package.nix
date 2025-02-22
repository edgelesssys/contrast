# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ buildGoModule }:

buildGoModule rec {
  pname = "tdx-measure";
  version = "0.1.0";

  # The source of the main module of this repo. We filter for Go files so that
  # changes in the other parts of this repo don't trigger a rebuild.
  src = ../../../tools/tdx-measure;

  proxyVendor = true;
  vendorHash = "sha256-Z5Au1Uye/yIm8N52LAvKe2EJ4PbJ382afaJxAV9C1SM=";

  subPackages = [ "." ];

  env.CGO_ENABLED = 0;

  ldflags = [
    "-s"
    "-X main.version=v${version}"
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
}
