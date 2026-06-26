# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ lib, buildGoModule }:

buildGoModule (finalAttrs: {
  pname = "collateral-proxy";
  version = builtins.readFile ../../../version.txt;

  src =
    let
      inherit (lib) fileset path hasSuffix;
      root = ../../../.;
    in
    fileset.toSource {
      inherit root;
      fileset = fileset.unions [
        (path.append root "collateral-proxy/go.mod")
        (path.append root "collateral-proxy/go.sum")
        (fileset.fileFilter (file: hasSuffix ".go" file.name) (path.append root "collateral-proxy"))
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-GrNc8vmx8p2Cb0FeGyeVKlxYve5KvhZpID/A9MLi/Sw=";

  sourceRoot = "${finalAttrs.src.name}/collateral-proxy";
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

  meta.mainProgram = "collateral-proxy";
})
