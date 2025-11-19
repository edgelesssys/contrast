# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ lib, buildGoModule }:

buildGoModule (finalAttrs: {
  pname = "imagestore";
  version = builtins.readFile ../../../version.txt;

  src =
    let
      inherit (lib) fileset path;
      root = ../../../.;
    in
    fileset.toSource {
      inherit root;
      fileset = fileset.unions [
        (path.append root "imagestore")
        (path.append root "internal/cryptsetup")
        (path.append root "go.mod")
        (path.append root "go.sum")
      ];
    };

  sourceRoot = "${finalAttrs.src.name}/imagestore";

  proxyVendor = true;
  vendorHash = "sha256-0WWk4i3AqwO1EaMFow8SW6XykuPTQ4m+Lyx74bn1TvM=";

  env.CGO_ENABLED = 0;
  dontFixup = true;

  ldflags = [
    "-s"
    "-X main.version=v${finalAttrs.version}"
  ];

  meta.mainProgram = "imagestore";
})
