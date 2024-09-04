# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  buildGoModule,
  buildGoTest,
  microsoft,
  kata,
  genpolicy ? microsoft.genpolicy,
  contrast,
  installShellFiles,
}:

let
  e2e = buildGoTest {
    inherit (contrast)
      version
      src
      proxyVendor
      vendorHash
      prePatch
      postPatch
      CGO_ENABLED
      ;
    pname = "${contrast.pname}-e2e";

    tags = [ "e2e" ];

    ldflags = [ "-s" ];

    subPackages = [
      "e2e/genpolicy"
      "e2e/getdents"
      "e2e/openssl"
      "e2e/servicemesh"
      "e2e/release"
      "e2e/policy"
      "e2e/workloadsecret"
      "e2e/volumestatefulset"
      "e2e/regression"
    ];
  };

  # Reference values that we embed into the Contrast CLI for
  # deployment generation and attestation.
  embeddedReferenceValues =
    let
      runtimeHandler =
        platform: hashFile:
        "contrast-cc-${platform}-${builtins.substring 0 8 (builtins.readFile hashFile)}";

      aks-clh-snp-handler = runtimeHandler "aks-clh-snp" microsoft.contrast-node-installer-image.runtimeHash;
      k3s-qemu-tdx-handler = runtimeHandler "k3s-qemu-tdx" kata.contrast-node-installer-image.runtimeHash;
      rke2-qemu-tdx-handler = runtimeHandler "rke2-qemu-tdx" kata.contrast-node-installer-image.runtimeHash;
      k3s-qemu-snp-handler = runtimeHandler "k3s-qemu-snp" kata.contrast-node-installer-image.runtimeHash;

      aksRefVals = {
        snp = [
          {
            minimumTCB = {
              bootloaderVersion = 3;
              teeVersion = 0;
              snpVersion = 8;
              microcodeVersion = 115;
            };
            trustedMeasurement = lib.removeSuffix "\n" (
              builtins.readFile (
                if microsoft.contrast-node-installer-image.debugRuntime then
                  (microsoft.kata-igvm.override { debug = true; }).launch-digest
                else
                  microsoft.kata-igvm.launch-digest
              )
            );
            productName = "Milan";
          }
        ];
      };

      snpRefVals = {
        snp =
          let
            launch-digest =
              if kata.contrast-node-installer-image.debugRuntime then
                kata.snp-launch-digest.override { debug = true; }
              else
                kata.snp-launch-digest;
          in
          [
            {
              trustedMeasurement = lib.removeSuffix "\n" (builtins.readFile "${launch-digest}/milan.hex");
              productName = "Milan";
            }
            {
              trustedMeasurement = lib.removeSuffix "\n" (builtins.readFile "${launch-digest}/genoa.hex");
              productName = "Genoa";
            }
          ];
      };

      fakeSha384Hash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
      tdxRefVals = {
        tdx = [
          {
            mrTd = fakeSha384Hash;
            rtrms = [
              fakeSha384Hash
              fakeSha384Hash
              fakeSha384Hash
              fakeSha384Hash
            ];
            minimumQeSvn = 0;
            minimumPceSvn = 0;
            # TODO(freax13): Remove this. We should ask the user to fill this in instead of providing our own defaults.
            minimumTeeTcbSvn = "04010200000000000000000000000000";
            # TODO(freax13): Remove this. We should ask the user to fill this in instead of providing our own defaults.
            mrSeam = "1cc6a17ab799e9a693fac7536be61c12ee1e0fabada82d0c999e08ccee2aa86de77b0870f558c570e7ffe55d6d47fa04";
            tdAttributes = "0000001000000000";
            xfam = "e702060000000000";
          }
        ];
      };
    in
    builtins.toFile "reference-values.json" (
      builtins.toJSON {
        "${aks-clh-snp-handler}" = aksRefVals;
        "${k3s-qemu-tdx-handler}" = tdxRefVals;
        "${rke2-qemu-tdx-handler}" = tdxRefVals;
        "${k3s-qemu-snp-handler}" = snpRefVals;
      }
    );

  packageOutputs = [
    "coordinator"
    "initializer"
    "cli"
    "nodeinstaller"
  ];
in

buildGoModule rec {
  pname = "contrast";
  version = builtins.readFile ../../../version.txt;

  outputs = packageOutputs ++ [ "out" ];

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
        (path.append root "cli/genpolicy/assets/allow-all.rego")
        (path.append root "internal/manifest/Milan.pem")
        (path.append root "internal/manifest/Genoa.pem")
        (path.append root "nodeinstaller")
        (path.append root "internal/attestation/tdx/Intel_SGX_Provisioning_Certification_RootCA.pem")
        (fileset.difference (fileset.fileFilter (file: hasSuffix ".go" file.name) root) (
          path.append root "service-mesh"
        ))
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-Ph3e6Gt1axYHCmDM4C4xXrZYexPBKqsQtshMPGWrQEE=";

  nativeBuildInputs = [ installShellFiles ];

  subPackages = packageOutputs ++ [ "internal/kuberesource/resourcegen" ];

  prePatch = ''
    install -D ${lib.getExe genpolicy} cli/genpolicy/assets/genpolicy
    install -D ${microsoft.genpolicy.rules}/genpolicy-rules.rego cli/genpolicy/assets/genpolicy-rules-microsoft.rego
    install -D ${kata.genpolicy.rules}/genpolicy-rules.rego cli/genpolicy/assets/genpolicy-rules-kata.rego
    install -D ${embeddedReferenceValues} internal/manifest/assets/reference-values.json
  '';

  # postPatch will be overwritten by the release-cli derivation, prePatch
  postPatch = ''
    install -D ${microsoft.genpolicy.settings-dev}/genpolicy-settings.json cli/genpolicy/assets/genpolicy-settings-microsoft.json
    install -D ${kata.genpolicy.settings-dev}/genpolicy-settings.json cli/genpolicy/assets/genpolicy-settings-kata.json
  '';

  CGO_ENABLED = 0;
  ldflags = [
    "-s"
    "-X github.com/edgelesssys/contrast/internal/constants.Version=${version}"
    "-X github.com/edgelesssys/contrast/internal/constants.MicrosoftGenpolicyVersion=${microsoft.genpolicy.version}"
    "-X github.com/edgelesssys/contrast/internal/constants.KataGenpolicyVersion=${kata.genpolicy.version}"
  ];

  preCheck = ''
    export CGO_ENABLED=1
  '';

  checkPhase = ''
    runHook preCheck
    go test -race ./...
    runHook postCheck
  '';

  postInstall = ''
    for sub in ${builtins.concatStringsSep " " packageOutputs}; do
      mkdir -p "''${!sub}/bin"
      mv "$out/bin/$sub" "''${!sub}/bin/$sub"
    done

    # rename the cli binary to contrast
    mv "$cli/bin/cli" "$cli/bin/contrast"

    # rename the nodeinstaller binary to node-installer
    mv "$nodeinstaller/bin/nodeinstaller" "$nodeinstaller/bin/node-installer"

    installShellCompletion --cmd contrast \
      --bash <($cli/bin/contrast completion bash) \
      --fish <($cli/bin/contrast completion fish) \
      --zsh <($cli/bin/contrast completion zsh)

    mkdir -p $cli/share
    mv $out/share/* $cli/share
  '';

  # Skip fixup as binaries are already stripped and we don't
  # need any other fixup, saving some seconds.
  dontFixup = true;

  passthru = {
    inherit e2e embeddedReferenceValues;
  };

  meta.mainProgram = "contrast";
}
