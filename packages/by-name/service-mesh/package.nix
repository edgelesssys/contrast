# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ lib, buildGoModule }:

buildGoModule (finalAttrs: {
  pname = "service-mesh";
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
        (path.append root "service-mesh/go.mod")
        (path.append root "service-mesh/go.sum")
        (path.append root "service-mesh/golden/defaultEnvoy.json")
        (fileset.fileFilter (file: hasSuffix ".go" file.name) (path.append root "internal/defaultdeny"))
        (fileset.fileFilter (file: hasSuffix ".go" file.name) (path.append root "internal/logger"))
        (fileset.fileFilter (file: hasSuffix ".go" file.name) (path.append root "service-mesh"))
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-fLYbI9SzxUbqmB9mM+CgKb6ka0/GSozJXacFe9baRiM=";

  sourceRoot = "${finalAttrs.src.name}/service-mesh";
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
