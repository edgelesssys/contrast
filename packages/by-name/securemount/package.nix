# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ buildGoModule }:

buildGoModule (finalAttrs: {
  pname = "securemount";
  version = builtins.readFile ../../../version.txt;

  src = ../../../.;

  subPackages = [ "securemount" ];

  proxyVendor = true;
  vendorHash = "sha256-fBQpmJg3n386LTZFI/08SXpGZm4+R8bz8QDlI0gf4jM=";

  env.CGO_ENABLED = 0;
  dontFixup = true;

  ldflags = [
    "-s"
    "-X main.version=v${finalAttrs.version}"
  ];

  meta.mainProgram = "securemount";
})
