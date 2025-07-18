# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ buildGoModule }:

buildGoModule (finalAttrs: {
  pname = "imagepuller";
  version = builtins.readFile ../../../version.txt;

  src = ../../../imagepuller;

  proxyVendor = true;
  vendorHash = "sha256-4ucDH4VDXGSUkSAtcn2/tH8xuB8GT28RD6PSqVQY7rk=";

  env.CGO_ENABLED = 0;
  dontFixup = true;

  ldflags = [
    "-s"
    "-X main.version=v${finalAttrs.version}"
  ];

  meta.mainProgram = "imagepuller";
})
