# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ lib, buildGoModule }:

buildGoModule (finalAttrs: {
  pname = "imagepuller";
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
        (path.append root "imagepuller/go.mod")
        (path.append root "imagepuller/go.sum")
        (fileset.fileFilter (file: hasSuffix ".go" file.name) (path.append root "imagepuller"))
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-4ucDH4VDXGSUkSAtcn2/tH8xuB8GT28RD6PSqVQY7rk=";

  sourceRoot = "${finalAttrs.src.name}/imagepuller";
  subPackages = [ "." ];

  env.CGO_ENABLED = 0;
  dontFixup = true;

  ldflags = [
    "-s"
    "-X main.version=v${finalAttrs.version}"
  ];

  meta.mainProgram = "imagepuller";
})
