# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ lib, buildGoModule }:

buildGoModule rec {
  pname = "igvm-go";
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
        (path.append root "tools/igvm/go.mod")
        (path.append root "tools/igvm/go.sum")
        (fileset.fileFilter (file: hasSuffix ".go" file.name) (path.append root "tools/igvm"))
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-QlOOmsYB4qK3Bgf+PB+QXs65XnwKz2ds3GJuvwCSc+k=";

  sourceRoot = "${src.name}/tools/igvm";
  subPackages = [ "cmd/igvm" ];

  env.CGO_ENABLED = 0;

  ldflags = [
    "-s"
    "-X main.version=v${version}"
  ];

  meta.mainProgram = "igvm";
}
