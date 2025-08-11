# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ buildGoModule }:

buildGoModule (finalAttrs: {
  pname = "securemount";
  version = builtins.readFile ../../../version.txt;

  src = ../../../securemount;

  proxyVendor = true;
  vendorHash = "sha256-T55Hs6Cn++8cnIu+yBtrY04fKUb+c7n7zsr2kML45Lg=";

  env.CGO_ENABLED = 0;
  dontFixup = true;

  ldflags = [
    "-s"
    "-X main.version=v${finalAttrs.version}"
  ];

  meta.mainProgram = "securemount";
})
