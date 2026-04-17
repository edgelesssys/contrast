# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{ lib, buildGoModule }:

buildGoModule (finalAttrs: {
  pname = "sev-snp-measure";
  version = "0.1.0";

  src =
    let
      inherit (lib) fileset path;
      root = ../../../../.;
    in
    fileset.toSource {
      inherit root;
      fileset = fileset.unions [
        (path.append root "tools/sev-snp-measure-go")
        (path.append root "internal/snp")
        (path.append root "go.mod")
        (path.append root "go.sum")
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-Qq31w1MyEsiTzj/2lUYCekTFqQDXtyiTttV8rvtpLGY=";

  sourceRoot = "${finalAttrs.src.name}/tools/sev-snp-measure-go";

  env.CGO_ENABLED = 0;

  ldflags = [ "-s" ];

  doCheck = false;

  meta.mainProgram = "sev-snp-measure-go";
})
