# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ lib, buildGoModule }:

buildGoModule (finalAttrs: {
  pname = "kds-proxy";
  version = builtins.readFile ../../../version.txt;

  src =
    let
      inherit (lib) fileset path hasSuffix;
      root = ../../../.;
    in
    fileset.toSource {
      inherit root;
      fileset = fileset.unions [
        (path.append root "kds-proxy/go.mod")
        (path.append root "kds-proxy/go.sum")
        (fileset.fileFilter (file: hasSuffix ".go" file.name) (path.append root "kds-proxy"))
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-UpUfi+SkMdjdz5xzIqfGQuPsdLC+D6mD9ObCgFeuuoQ=";

  sourceRoot = "${finalAttrs.src.name}/kds-proxy";
  subPackages = [ "." ];

  env.CGO_ENABLED = 0;

  ldflags = [
    "-s"
    "-X main.version=v${finalAttrs.version}"
  ];

  checkPhase = ''
    runHook preCheck
    go test ./...
    runHook postCheck
  '';

  meta.mainProgram = "kds-proxy";
})
