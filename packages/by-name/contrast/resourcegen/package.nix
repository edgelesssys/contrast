# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  buildGoModule,
  reference-values,
}:

buildGoModule (_finalAttrs: {
  pname = "contrast-resourcegen";
  version = builtins.readFile ../../../../version.txt;

  outputs = [ "out" ];

  # The source of the main module of this repo. We filter for Go files so that
  # changes in the other parts of this repo don't trigger a rebuild.
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
        (fileset.fileFilter (file: hasSuffix ".yaml" file.name) (
          path.append root "internal/kuberesource/assets"
        ))
        (path.append root "internal/manifest/Milan.pem")
        (path.append root "internal/manifest/Genoa.pem")
        (path.append root "internal/manifest/Intel_SGX_Provisioning_Certification_RootCA.pem")
        (fileset.intersection (fileset.fileFilter (file: hasSuffix ".go" file.name) root) (
          fileset.unions [
            (path.append root "internal/attestation")
            (path.append root "internal/constants")
            (path.append root "internal/idblock")
            (path.append root "internal/kuberesource")
            (path.append root "internal/manifest")
            (path.append root "internal/platforms")
            (path.append root "internal/userapi")
          ]
        ))
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-xU4M0DSB/Pca7MqYAp7A3dPr2BsEtp8uAyxtM7yiMbs=";

  subPackages = [ "internal/kuberesource/resourcegen" ];

  prePatch = ''
    install -D ${reference-values} internal/manifest/assets/reference-values.json
  '';

  env.CGO_ENABLED = 0;

  ldflags = [
    "-s"
  ];

  tags = [ "contrast_unstable_api" ];

  preCheck = ''
    export CGO_ENABLED=1
  '';

  checkPhase = ''
    runHook preCheck
    runHook postCheck
  '';

  # Skip fixup as binaries are already stripped and we don't
  # need any other fixup, saving some seconds.
  dontFixup = true;

  meta = {
    description = "Resource generator for Contrast";
    mainProgram = "resourcegen";
  };
})
