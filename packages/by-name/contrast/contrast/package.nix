# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  buildGoModule,
  kata,
  installShellFiles,
  calculateSnpIDBlock,
  contrastPkgsStatic,
  OVMF-TDX,
}:

let
  # Reference values that we embed into the Contrast CLI for
  # deployment generation and attestation.
  embeddedReferenceValues =
    let
      runtimeHandler =
        platform: hashFile:
        "contrast-cc-${platform}-${builtins.substring 0 8 (builtins.readFile hashFile)}";

      metal-qemu-tdx-handler = runtimeHandler "metal-qemu-tdx" kata.contrast-node-installer-image.runtimeHash;
      metal-qemu-snp-handler = runtimeHandler "metal-qemu-snp" kata.contrast-node-installer-image.runtimeHash;
      metal-qemu-snp-gpu-handler = runtimeHandler "metal-qemu-snp-gpu" kata.contrast-node-installer-image.runtimeHash;
      metal-qemu-tdx-gpu-handler = runtimeHandler "metal-qemu-tdx-gpu" kata.contrast-node-installer-image.runtimeHash;

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

      tdxRefValsWith =
        {
          os-image,
          ovmf,
          withGPU,
        }:
        {
          tdx = [
            (
              let
                launch-digests = kata.calculateTdxLaunchDigests {
                  inherit os-image ovmf withGPU;
                  debug = kata.contrast-node-installer-image.debugRuntime;
                };
              in
              {
                mrTd = builtins.readFile "${launch-digests}/mrtd.hex";
                rtmrs = [
                  (builtins.readFile "${launch-digests}/rtmr0.hex")
                  (builtins.readFile "${launch-digests}/rtmr1.hex")
                  (builtins.readFile "${launch-digests}/rtmr2.hex")
                  (builtins.readFile "${launch-digests}/rtmr3.hex")
                ];
                xfam = "e702060000000000";
              }
            )
          ];
        };
      tdxRefVals = tdxRefValsWith {
        inherit (kata.contrast-node-installer-image) os-image;
        ovmf = OVMF-TDX;
        withGPU = false;
      };
      tdxGpuRefVals = tdxRefValsWith {
        inherit (kata.contrast-node-installer-image.gpu) os-image;
        ovmf = OVMF-TDX.override {
          # Only enable ACPI verification for the GPU build, until
          # the verification is actually secure.
          withACPIVerificationInsecure = true;
        };
        withGPU = true;
      };
    in
    builtins.toFile "reference-values.json" (
      builtins.toJSON {
        "${metal-qemu-tdx-handler}" = tdxRefVals;
        "${metal-qemu-snp-handler}" = snpRefVals;
        "${metal-qemu-snp-gpu-handler}" = snpGpuRefVals;
        "${metal-qemu-tdx-gpu-handler}" = tdxGpuRefVals;
      }
    );

  snpIdBlocksFor =
    os-image:
    let
      guestPolicy = builtins.fromJSON (builtins.readFile ./snpGuestPolicyQEMU.json);
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
        inherit guestPolicy;
      };
      Genoa = {
        idBlock = builtins.readFile "${idBlocks}/id-block-genoa.base64";
        idAuth = builtins.readFile "${idBlocks}/id-auth-genoa.base64";
        inherit guestPolicy;
      };
    };
  snpIdBlocks = builtins.toFile "snp-id-blocks.json" (
    builtins.toJSON {
      metal-qemu-snp = snpIdBlocksFor kata.contrast-node-installer-image.os-image;
      metal-qemu-snp-gpu = snpIdBlocksFor kata.contrast-node-installer-image.gpu.os-image;
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
        (path.append root "cli/cmd/assets/image-replacements.txt")
        (fileset.fileFilter (file: hasSuffix ".yaml" file.name) (
          path.append root "internal/kuberesource/assets"
        ))
        (path.append root "internal/manifest/Milan.pem")
        (path.append root "internal/manifest/Genoa.pem")
        (path.append root "internal/manifest/Intel_SGX_Provisioning_Certification_RootCA.pem")
        (path.append root "nodeinstaller")
        (fileset.difference (fileset.fileFilter (file: hasSuffix ".go" file.name) root) (
          fileset.unions [
            (path.append root "service-mesh")
            (path.append root "tools")
            (path.append root "imagepuller")
            (path.append root "imagestore")
            (path.append root "initdata-processor")
            (path.append root "e2e")
          ]
        ))
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-G6NiezmeQ2Gq1ZrrZxgtlECP5MNbQWTojLAXcl18Ofo=";

  nativeBuildInputs = [ installShellFiles ];

  subPackages = packageOutputs ++ [ "internal/kuberesource/resourcegen" ];

  prePatch = ''
    install -D ${lib.getExe contrastPkgsStatic.kata.genpolicy} cli/genpolicy/assets/genpolicy-kata
    install -D ${kata.genpolicy.rules}/genpolicy-rules.rego cli/genpolicy/assets/genpolicy-rules-kata.rego
    install -D ${embeddedReferenceValues} internal/manifest/assets/reference-values.json
    install -D ${snpIdBlocks} nodeinstaller/internal/kataconfig/snp-id-blocks.json
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
    inherit embeddedReferenceValues;
  };

  meta.mainProgram = "contrast";
})
