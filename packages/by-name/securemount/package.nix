# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ buildGoModule }:

buildGoModule (finalAttrs: {
  pname = "securemount";
  version = builtins.readFile ../../../version.txt;

  src = ../../../securemount;

  proxyVendor = true;
  vendorHash = "sha256-18XveHCYkwktiGEUccF435hzIzoYKM/3PT7axU6oVRo=";

  env.CGO_ENABLED = 0;
  dontFixup = true;

  ldflags = [
    "-s"
    "-X main.version=v${finalAttrs.version}"
  ];

  meta.mainProgram = "securemount";
})
