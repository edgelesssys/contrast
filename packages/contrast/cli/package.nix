# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  buildGoModule,
  contrast,
  kata,
  installShellFiles,
  contrastPkgsStatic,
  reference-values,
}:

buildGoModule (finalAttrs: {
  pname = "${contrast.pname}-cli";
  inherit (contrast)
    version
    proxyVendor
    vendorHash
    ;

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
        (path.append root "go.mod")
        (path.append root "go.sum")
        (path.append root "cli/cmd/assets/image-replacements.txt")
        (fileset.fileFilter (file: hasSuffix ".yaml" file.name) (
          path.append root "internal/kuberesource/assets"
        ))
        (path.append root "internal/manifest/Milan.pem")
        (path.append root "internal/manifest/Genoa.pem")
        (path.append root "internal/manifest/Intel_SGX_Provisioning_Certification_RootCA.pem")
        (fileset.intersection (fileset.fileFilter (file: hasSuffix ".go" file.name) root) (
          fileset.unions [
            (path.append root "internal")
            (path.append root "cli")
            (path.append root "sdk")
          ]
        ))
      ];
    };

  subPackages = [ "cli" ];

  nativeBuildInputs = [ installShellFiles ];

  prePatch = ''
    install -D ${lib.getExe contrastPkgsStatic.kata.genpolicy} cli/genpolicy/assets/genpolicy-kata
    install -D ${kata.genpolicy.rules}/genpolicy-rules.rego cli/genpolicy/assets/genpolicy-rules-kata.rego
    install -D ${reference-values} internal/manifest/assets/reference-values.json
  '';

  # postPatch will be overwritten by the release-cli derivation, prePatch won't.
  postPatch = ''
    install -D ${kata.genpolicy.settings-dev}/genpolicy-settings.json cli/genpolicy/assets/genpolicy-settings-kata.json
  '';

  env.CGO_ENABLED = 0;

  ldflags = [
    "-s"
    "-X github.com/edgelesssys/contrast/internal/constants.Version=v${finalAttrs.version}"
    "-X github.com/edgelesssys/contrast/internal/constants.KataGenpolicyVersion=${kata.genpolicy.version}"
  ];

  tags = [ "contrast_unstable_api" ];

  preCheck = ''
    export CGO_ENABLED=1
  '';

  checkPhase = ''
    runHook preCheck
    go test -tags=${lib.concatStringsSep "," finalAttrs.tags} -race ./cli/...
    runHook postCheck
  '';

  postInstall = ''
    # rename the cli binary to contrast
    mv "$out/bin/cli" "$out/bin/contrast"

    installShellCompletion --cmd contrast \
      --bash <($out/bin/contrast completion bash) \
      --fish <($out/bin/contrast completion fish) \
      --zsh <($out/bin/contrast completion zsh)
  '';

  # Skip fixup as binaries are already stripped and we don't
  # need any other fixup, saving some seconds.
  dontFixup = true;

  meta.mainProgram = "contrast";
})
