# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ lib, buildGoModule }:

buildGoModule (finalAttrs: {
  pname = "initdata-processor";
  version = builtins.readFile ../../../version.txt;

  # The source of the main module of this repo. We filter for Go files so that
  # changes in the other parts of this repo don't trigger a rebuild.
  src =
    let
      inherit (lib) fileset path hasSuffix;
      root = ../../../.;
    in
    fileset.toSource {
      inherit root;
      fileset = fileset.unions [
        (path.append root "go.mod")
        (path.append root "go.sum")
        (path.append root "initdata-processor/go.mod")
        (path.append root "initdata-processor/go.sum")
        (fileset.fileFilter (file: hasSuffix ".go" file.name) (path.append root "internal/initdata"))
        (fileset.fileFilter (file: hasSuffix ".go" file.name) (path.append root "initdata-processor"))
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-LhwTUKR8x6J3CcWrHqMKjLUzOzcPPMXIP8n+yTta5V4=";

  sourceRoot = "${finalAttrs.src.name}/initdata-processor";
  subPackages = [ "." ];

  env.CGO_ENABLED = 0;

  ldflags = [
    "-s"
    "-X main.version=v${finalAttrs.version}"
  ];

  preCheck = ''
    export CGO_ENABLED=1
  '';

  checkPhase = ''
    runHook preCheck
    go test -race ./...
    runHook postCheck
  '';

  meta.mainProgram = "initdata-processor";
})
