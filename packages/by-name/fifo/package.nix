# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ buildGoModule }:

buildGoModule {
  pname = "fifo";
  version = "0.1.0";

  src = ../../../tools/fifo;

  proxyVendor = true;
  vendorHash = "sha256-TBXaMzHkH76g2zOwAzEsVxo5u+1QyNJBnsQEivsgYRg=";

  env.CGO_ENABLED = 0;

  ldflags = [ "-s" ];

  meta.mainProgram = "fifo";
}
