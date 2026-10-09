# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  buildGoModule,
}:

buildGoModule {
  pname = "fifo";
  version = "0.1.0";

  src = ../../../tools/fifo;

  proxyVendor = true;
  vendorHash = "sha256-B7SL+UcKUiLD736CDQcJc2Adpdbcs7NACeoo9V5DfEE=";

  env.CGO_ENABLED = 0;

  ldflags = [ "-s" ];

  meta = lib.contrast.ourMeta { mainProgram = "fifo"; };
}
