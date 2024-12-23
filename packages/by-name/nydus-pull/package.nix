# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ buildGoModule }:

buildGoModule rec {
  pname = "nydus-pull";
  version = builtins.readFile ../../../version.txt;

  src = ../../../tools/nydus-pull;

  proxyVendor = true;
  vendorHash = "sha256-bzCdcDfdivf52CerJ+9Nf5i+/laqjBWKNhhyLS8eBs4=";

  subPackages = [ "." ];

  CGO_ENABLED = 0;
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

  meta.mainProgram = "nydus-pull";
}
