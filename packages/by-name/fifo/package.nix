# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ buildGoModule }:

buildGoModule {
  pname = "fifo";
  version = "0.1.0";

  src = ../../../tools/fifo;

  proxyVendor = true;
  vendorHash = "sha256-R4ObM2ZSD/lFQQNY1fKuRZ8zoNy2j8t6RJnJOf4v/7E=";

  env.CGO_ENABLED = 0;

  ldflags = [ "-s" ];

  meta.mainProgram = "fifo";
}
