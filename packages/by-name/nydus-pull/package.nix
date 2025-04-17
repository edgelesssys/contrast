# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ buildGoModule }:

buildGoModule (finalAttrs: {
  pname = "nydus-pull";
  version = builtins.readFile ../../../version.txt;

  src = ../../../tools/nydus-pull;

  proxyVendor = true;
  vendorHash = "sha256-2w70zc0VRUbWNrx7ZubbIB+tOaeQBKbEZIgWROprmeg=";

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

  meta.mainProgram = "nydus-pull";
})
