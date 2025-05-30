# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  buildGoModule,
  buildGoTest,
  microsoft,
  kata,
  contrast,
  installShellFiles,
  calculateSnpIDBlock,
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
      ;
    pname = "${contrast.pname}-e2e";

    tags = contrast.tags ++ [ "e2e" ];

    env.CGO_ENABLED = 0;

    subPackages = [
      # keep-sorted start
      "e2e/aks-runtime"
      "e2e/atls"
      "e2e/genpolicy"
      "e2e/getdents"
      "e2e/gpu"
      "e2e/multiple-cpus"
      "e2e/openssl"
      "e2e/peerrecovery"
      "e2e/policy"
      "e2e/regression"
      "e2e/release"
      "e2e/servicemesh"
      "e2e/volumestatefulset"
      "e2e/workloadsecret"
      # keep-sorted end
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
      metal-qemu-tdx-handler = runtimeHandler "metal-qemu-tdx" kata.contrast-node-installer-image.runtimeHash;
      k3s-qemu-tdx-handler = runtimeHandler "k3s-qemu-tdx" kata.contrast-node-installer-image.runtimeHash;
      rke2-qemu-tdx-handler = runtimeHandler "rke2-qemu-tdx" kata.contrast-node-installer-image.runtimeHash;
      metal-qemu-snp-handler = runtimeHandler "metal-qemu-snp" kata.contrast-node-installer-image.runtimeHash;
      metal-qemu-snp-gpu-handler = runtimeHandler "metal-qemu-snp-gpu" kata.contrast-node-installer-image.runtimeHash;
      k3s-qemu-snp-handler = runtimeHandler "k3s-qemu-snp" kata.contrast-node-installer-image.runtimeHash;
      k3s-qemu-snp-gpu-handler = runtimeHandler "k3s-qemu-snp-gpu" kata.contrast-node-installer-image.runtimeHash;
      aksRefVals = {
        snp = [
          {
            guestPolicy = builtins.fromJSON (builtins.readFile microsoft.kata-igvm.snp-guest-policy);
            platformInfo = {
              SMTEnabled = true;
            };
            minimumTCB = {
              bootloaderVersion = 3;
              teeVersion = 0;
              snpVersion = 8;
              microcodeVersion = 115;
            };
            trustedMeasurement = builtins.readFile (
              if microsoft.contrast-node-installer-image.debugRuntime then
                "${(microsoft.kata-igvm.override { debug = true; }).snp-launch-digest}/milan.hex"
              else
                "${microsoft.kata-igvm.snp-launch-digest}/milan.hex"
            );
            productName = "Milan";
          }
        ];
      };

      snpRefValsWith = os-image: {
        snp =
          let
            guestPolicy = builtins.fromJSON (builtins.readFile ./snpGuestPolicyQEMU.json);
            platformInfo = {
              SMTEnabled = true;
            };
            launch-digest = kata.calculateSnpLaunchDigest {
              inherit os-image;
              debug = kata.contrast-node-installer-image.debugRuntime;
            };
          in
          [
            {
              inherit guestPolicy platformInfo;
              trustedMeasurement = builtins.readFile "${launch-digest}/milan.hex";
              productName = "Milan";
            }
            {
              inherit guestPolicy platformInfo;
              trustedMeasurement = builtins.readFile "${launch-digest}/genoa.hex";
              productName = "Genoa";
            }
          ];
      };

      snpRefVals = snpRefValsWith kata.contrast-node-installer-image.os-image;
      snpGpuRefVals = snpRefValsWith kata.contrast-node-installer-image.gpu.os-image;

      tdxRefVals = {
        tdx = [
          (
            let
              launch-digests = kata.calculateTdxLaunchDigests {
                inherit (kata.contrast-node-installer-image) os-image;
                debug = kata.contrast-node-installer-image.debugRuntime;
              };
            in
            {
              mrTd = builtins.readFile "${launch-digests}/mrtd.hex";
              rtrms = [
                (builtins.readFile "${launch-digests}/rtmr0.hex")
                (builtins.readFile "${launch-digests}/rtmr1.hex")
                (builtins.readFile "${launch-digests}/rtmr2.hex")
                (builtins.readFile "${launch-digests}/rtmr3.hex")
              ];
              minimumQeSvn = 0;
              minimumPceSvn = 0;
              tdAttributes = "0000001000000000";
              xfam = "e702060000000000";
            }
          )
        ];
      };
    in
    builtins.toFile "reference-values.json" (
      builtins.toJSON {
        "${aks-clh-snp-handler}" = aksRefVals;
        "${metal-qemu-tdx-handler}" = tdxRefVals;
        "${k3s-qemu-tdx-handler}" = tdxRefVals;
        "${rke2-qemu-tdx-handler}" = tdxRefVals;
        "${metal-qemu-snp-handler}" = snpRefVals;
        "${metal-qemu-snp-gpu-handler}" = snpGpuRefVals;
        "${k3s-qemu-snp-handler}" = snpRefVals;
        "${k3s-qemu-snp-gpu-handler}" = snpGpuRefVals;
      }
    );

  snpIdBlocksFor =
    os-image:
    let
      launch-digest = kata.calculateSnpLaunchDigest {
        inherit os-image;
        debug = kata.contrast-node-installer-image.debugRuntime;
      };
      idBlocks = calculateSnpIDBlock {
        snp-launch-digest = launch-digest;
        snp-guest-policy = ./snpGuestPolicyQEMU.json;
      };
    in
    {
      Milan = {
        idBlock = builtins.readFile "${idBlocks}/id-block-milan.base64";
        idAuth = builtins.readFile "${idBlocks}/id-auth-milan.base64";
      };
      Genoa = {
        idBlock = builtins.readFile "${idBlocks}/id-block-genoa.base64";
        idAuth = builtins.readFile "${idBlocks}/id-auth-genoa.base64";
      };
    };
  snpIdBlocks = builtins.toFile "snp-id-blocks.json" (
    builtins.toJSON {
      metal-qemu-snp = snpIdBlocksFor kata.contrast-node-installer-image.os-image;
      metal-qemu-snp-gpu = snpIdBlocksFor kata.contrast-node-installer-image.gpu.os-image;
      k3s-qemu-snp = snpIdBlocksFor kata.contrast-node-installer-image.os-image;
      k3s-qemu-snp-gpu = snpIdBlocksFor kata.contrast-node-installer-image.gpu.os-image;
      aks-clh-snp.Milan =
        let
          launch-digest =
            if microsoft.contrast-node-installer-image.debugRuntime then
              (microsoft.kata-igvm.override { debug = true; }).snp-launch-digest
            else
              microsoft.kata-igvm.snp-launch-digest;
          idBlocks = calculateSnpIDBlock {
            snp-launch-digest = launch-digest;
            inherit (microsoft.kata-igvm) snp-guest-policy;
          };
        in
        {
          idBlock = builtins.readFile "${idBlocks}/id-block-milan.base64";
          idAuth = builtins.readFile "${idBlocks}/id-auth-milan.base64";
        };
    }
  );

  packageOutputs = [
    "coordinator"
    "initializer"
    "cli"
    "nodeinstaller"
  ];
in

buildGoModule (finalAttrs: {
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
        (path.append root "internal/manifest/Intel_SGX_Provisioning_Certification_RootCA.pem")
        (path.append root "nodeinstaller")
        (fileset.difference (fileset.fileFilter (file: hasSuffix ".go" file.name) root) (
          fileset.unions [
            (path.append root "service-mesh")
            (path.append root "tools")
          ]
        ))
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-iX+eMVSEfFAtEVCRmtM8zyvbA6Jeix0ybiUJSXYe1dU=";

  nativeBuildInputs = [ installShellFiles ];

  subPackages = packageOutputs ++ [ "internal/kuberesource/resourcegen" ];

  prePatch = ''
    install -D ${lib.getExe microsoft.genpolicy} cli/genpolicy/assets/genpolicy-microsoft
    install -D ${lib.getExe kata.genpolicy} cli/genpolicy/assets/genpolicy-kata
    install -D ${microsoft.genpolicy.rules}/genpolicy-rules.rego cli/genpolicy/assets/genpolicy-rules-microsoft.rego
    install -D ${kata.genpolicy.rules}/genpolicy-rules.rego cli/genpolicy/assets/genpolicy-rules-kata.rego
    install -D ${embeddedReferenceValues} internal/manifest/assets/reference-values.json
    install -D ${snpIdBlocks} nodeinstaller/internal/constants/snp-id-blocks.json
  '';

  # postPatch will be overwritten by the release-cli derivation, prePatch won't.
  postPatch = ''
    install -D ${microsoft.genpolicy.settings-dev}/genpolicy-settings.json cli/genpolicy/assets/genpolicy-settings-microsoft.json
    install -D ${kata.genpolicy.settings-dev}/genpolicy-settings.json cli/genpolicy/assets/genpolicy-settings-kata.json
  '';

  env.CGO_ENABLED = 0;

  ldflags = [
    "-s"
    "-X github.com/edgelesssys/contrast/internal/constants.Version=v${finalAttrs.version}"
    "-X github.com/edgelesssys/contrast/internal/constants.MicrosoftGenpolicyVersion=${microsoft.genpolicy.version}"
    "-X github.com/edgelesssys/contrast/internal/constants.KataGenpolicyVersion=${kata.genpolicy.version}"
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
})
