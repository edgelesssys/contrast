# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ buildGoModule }:

buildGoModule {
  pname = "sbom-generator";
  version = "0.1.0";

  src = ../../../tools/sbom-generator;

  vendorHash = null;

  subPackages = [ "." ];

  env.CGO_ENABLED = 0;

  ldflags = [ "-s" ];

  meta.mainProgram = "sbom-generator";
}
