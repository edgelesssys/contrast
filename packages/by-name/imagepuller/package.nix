# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ lib, buildGoModule }:

buildGoModule (finalAttrs: {
  pname = "imagepuller";
  version = builtins.readFile ../../../version.txt;

  src =
    let
      inherit (lib) fileset path;
      root = ../../../.;
    in
    fileset.toSource {
      inherit root;
      fileset = fileset.unions [
        (path.append root "imagepuller")
        (path.append root "internal/katacomponents")
        (path.append root "go.mod")
        (path.append root "go.sum")
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-lqT2fOJIuJAYRj0K++FLjS1hZVRplR81zc0tPRwqlOk=";

  sourceRoot = "${finalAttrs.src.name}/imagepuller";

  env.CGO_ENABLED = 0;
  dontFixup = true;

  ldflags = [
    "-s"
    "-X main.version=v${finalAttrs.version}"
  ];

  meta.mainProgram = "imagepuller";
})
