# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  buildGoModule,
  bash,
}:

buildGoModule {
  pname = "debugshell";
  version = "0.1.0";

  src =
    let
      inherit (lib) fileset path hasSuffix;
      root = ../../../tools/debugshell;
    in
    fileset.toSource {
      inherit root;
      fileset = fileset.unions [
        (path.append root "go.mod")
        (path.append root "go.sum")
        (fileset.fileFilter (file: hasSuffix ".go" file.name) root)
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-WYFvmqFkvr3YHwggvhqMyl0gyzUClEMRpkVGCCmmS7Q=";

  subPackages = [ "." ];

  env.CGO_ENABLED = 0;

  ldflags = [
    "-s"
    "-X main.bashPath=${lib.getExe bash}"
  ];

  meta = lib.contrast.ourMeta { mainProgram = "debugshell"; };
}
