# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ lib, buildGoModule }:

buildGoModule (finalAttrs: {
  pname = "service-mesh";
  version = builtins.readFile ../../../version.txt;

  # The source of the main module of this repo. We filter for Go files so that
  # changes in the other parts of this repo don't trigger a rebuild.
  src =
    let
      inherit (lib) fileset path hasSuffix;
      root = ../../../service-mesh;
    in
    fileset.toSource {
      inherit root;
      fileset = fileset.unions [
        (path.append root "go.mod")
        (path.append root "go.sum")
        (path.append root "golden/defaultEnvoy.json")
        (fileset.fileFilter (file: hasSuffix ".go" file.name) root)
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-f/n8jMfHyfukxqv+i9K8Arko+ztzIYDBlBSTSxtRd0w=";

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

  meta.mainProgram = "service-mesh";
})
