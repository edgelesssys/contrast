# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  buildGoModule,
  contrast,
}:

buildGoModule {
  buildTestBinaries = true;

  pname = "${contrast.pname}-e2e";
  inherit (contrast)
    version
    proxyVendor
    vendorHash
    prePatch
    postPatch
    ;

  src =
    let
      inherit (lib) fileset path hasSuffix;
      root = ../../../../.;
      toRootPath = p: path.append root p;
    in
    fileset.toSource {
      inherit root;
      fileset = fileset.unions (
        (lib.map toRootPath [
          "go.mod"
          "go.sum"
          "cli/cmd/assets/image-replacements.txt"
          "internal/manifest/Milan.pem"
          "internal/manifest/Genoa.pem"
          "internal/manifest/Intel_SGX_Provisioning_Certification_RootCA.pem"
        ])
        ++ [
          (fileset.intersection (fileset.fileFilter (file: hasSuffix ".go" file.name) root) (
            fileset.unions (
              lib.map toRootPath [
                "e2e"
                "internal"
                "cli"
                "sdk"
              ]
            )
          ))
        ]
      );
    };

  tags = contrast.tags ++ [ "e2e" ];

  env.CGO_ENABLED = 0;

  subPackages = [
    # keep-sorted start
    "e2e/atls"
    "e2e/attestation"
    "e2e/genpolicy-unsupported"
    "e2e/gpu"
    "e2e/imagepuller-auth"
    "e2e/imagestore"
    "e2e/memdump"
    "e2e/multiple-cpus"
    "e2e/openssl"
    "e2e/peerrecovery"
    "e2e/policy"
    "e2e/proxy"
    "e2e/regression"
    "e2e/release"
    "e2e/servicemesh"
    "e2e/vault"
    "e2e/volumestatefulset"
    "e2e/workloadsecret"
    # keep-sorted end
  ];
}
