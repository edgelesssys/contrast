# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ buildGoModule }:

buildGoModule (finalAttrs: {
  pname = "securemount";
  version = builtins.readFile ../../../version.txt;

  src = ../../../securemount;

  proxyVendor = true;
  vendorHash = "sha256-Wai/5v7XbtF6puxCYye0b9Iy3NrgBAa8lkbahA0Mk90=";

  env.CGO_ENABLED = 0;
  dontFixup = true;

  ldflags = [
    "-s"
    "-X main.version=v${finalAttrs.version}"
  ];

  meta.mainProgram = "securemount";
})
