# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  buildGoModule,
  reference-values,
}:

let
  packageOutputs = [
    "coordinator"
    "initializer"
  ];
in

buildGoModule (finalAttrs: {
  pname = "contrast";
  version = builtins.readFile ../../../../version.txt;

  outputs = packageOutputs ++ [ "out" ];

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
            (path.append root "internal")
            (path.append root "coordinator")
            (path.append root "initializer")
            (path.append root "sdk")
          ]
        ))
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-Nd17FKpYeU5ln40ltlACB4jhuXD7xPXOxQnqNUW01Pc=";

  prePatch = ''
    install -D ${reference-values} internal/manifest/assets/reference-values.json
  '';

  env.CGO_ENABLED = 0;

  ldflags = [
    "-s"
    "-X github.com/edgelesssys/contrast/internal/constants.Version=v${finalAttrs.version}"
  ];

  tags = [ "contrast_unstable_api" ];

  preCheck = ''
    export CGO_ENABLED=1
  '';

  checkPhase = ''
    runHook preCheck
    go test -tags=${lib.concatStringsSep "," finalAttrs.tags} -race ./...
    runHook postCheck
  '';

  postInstall = ''
    for sub in ${builtins.concatStringsSep " " packageOutputs}; do
      mkdir -p "''${!sub}/bin"
      mv "$out/bin/$sub" "''${!sub}/bin/$sub"
    done
  '';

  # Skip fixup as binaries are already stripped and we don't
  # need any other fixup, saving some seconds.
  dontFixup = true;
})
