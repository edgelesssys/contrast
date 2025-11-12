# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  buildGoModule,
  contrast,
}:

buildGoModule {
  buildTestBinaries = true;

  src =
    let
      inherit (lib) fileset path hasSuffix;
      root = ../../../../.;
    in
    fileset.toSource {
      inherit root;
      fileset = fileset.unions [
        (path.append root "go.mod")
        (path.append root "go.sum")
        (path.append root "cli/cmd/assets/image-replacements.txt")
        (path.append root "internal/manifest/Milan.pem")
        (path.append root "internal/manifest/Genoa.pem")
        (path.append root "internal/manifest/Intel_SGX_Provisioning_Certification_RootCA.pem")
        (fileset.difference (fileset.fileFilter (file: hasSuffix ".go" file.name) root) (
          fileset.unions [
            (path.append root "imagepuller")
            (path.append root "imagestore")
            (path.append root "initdata-processor")
            (path.append root "service-mesh")
            (path.append root "tools")
          ]
        ))
      ];
    };

  inherit (contrast)
    version
    proxyVendor
    vendorHash
    prePatch
    postPatch
    ;
  pname = "${contrast.pname}-e2e";

  tags = contrast.tags ++ [ "e2e" ];

  env.CGO_ENABLED = 0;

  subPackages = [
    # keep-sorted start
    "e2e/atls"
    "e2e/genpolicy-unsupported"
    "e2e/gpu"
    "e2e/imagestore"
    "e2e/memdump"
    "e2e/multiple-cpus"
    "e2e/openssl"
    "e2e/peerrecovery"
    "e2e/policy"
    "e2e/regression"
    "e2e/release"
    "e2e/servicemesh"
    "e2e/vault"
    "e2e/volumestatefulset"
    "e2e/workloadsecret"
    # keep-sorted end
  ];
}
