# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ buildGoModule }:

buildGoModule rec {
  pname = "tdx-measure";
  version = "0.1.0";

  src = ../../../tools/tdx-measure;

  proxyVendor = true;
  vendorHash = "sha256-Ufg+5ad0ThjBGDCOevvT0eKDGYFRs0mdG57EudpUM04=";

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
