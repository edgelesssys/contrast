# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  buildGoModule,
  contrast,
}:

buildGoModule (finalAttrs: {
  pname = "contrast-node-installer";
  version = builtins.readFile ../../../../version.txt;

  inherit (contrast)
    proxyVendor
    vendorHash
    ldflags
    tags
    preCheck
    dontFixup
    ;

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
        (path.append root "nodeinstaller")
        (path.append root "internal/manifest/Milan.pem")
        (path.append root "internal/manifest/Genoa.pem")
        (path.append root "internal/manifest/Intel_SGX_Provisioning_Certification_RootCA.pem")
        (fileset.fileFilter (file: hasSuffix ".go" file.name) (path.append root "internal"))
      ];
    };

  subPackages = [ "nodeinstaller" ];

  prePatch = ''
    install -D ${contrast.embeddedReferenceValues} internal/manifest/assets/reference-values.json
    install -D ${contrast.snpIdBlocks} nodeinstaller/internal/kataconfig/snp-id-blocks.json
  '';

  env.CGO_ENABLED = 0;

  checkPhase = ''
    runHook preCheck
    go test -tags=${lib.concatStringsSep "," finalAttrs.tags} -race ./nodeinstaller/...
    runHook postCheck
  '';

  postInstall = ''
    mv "$out/bin/nodeinstaller" "$out/bin/node-installer"
  '';

  meta.mainProgram = "node-installer";
})
