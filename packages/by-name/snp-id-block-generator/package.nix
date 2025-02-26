# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ lib, buildGoModule }:

buildGoModule rec {
  pname = "snp-id-block-generator";
  version = "0.1.0";

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
        (path.append root "tools/snp-id-block-generator/go.mod")
        (path.append root "tools/snp-id-block-generator/go.sum")
        (fileset.fileFilter (file: hasSuffix ".go" file.name) (path.append root "internal/idblock"))
        (fileset.fileFilter (file: hasSuffix ".go" file.name) (path.append root "internal/constants"))
        (fileset.fileFilter (file: hasSuffix ".go" file.name) (
          path.append root "tools/snp-id-block-generator"
        ))
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-fmn5WbWskttvzaNSo85nAv0P/sCjQ1HcIcWW7Swmzt8=";

  sourceRoot = "${src.name}/tools/snp-id-block-generator";
  subPackages = [ "." ];

  env.CGO_ENABLED = 0;

  ldflags = [
    "-s"
    "-X main.version=v${version}"
  ];

  preCheck = ''
    export CGO_ENABLED=1
  '';

  checkPhase = ''
    runHook preCheck
    go test -race ./...
    runHook postCheck
  '';

  meta.mainProgram = "snp-id-block-generator";
}
