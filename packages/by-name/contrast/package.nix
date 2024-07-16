# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  buildGoModule,
  buildGoTest,
  buildGoCache,
  microsoft,
  genpolicy ? microsoft.genpolicy,
  contrast,
  installShellFiles,
}:

let
  goCache = buildGoCache {
    importPackagesFile = ./imported-packages;
    inherit (contrast) vendorHash proxyVendor src;
  };
  e2e = buildGoTest {
    inherit (contrast)
      version
      src
      proxyVendor
      vendorHash
      prePatch
      CGO_ENABLED
      ;
    pname = "${contrast.pname}-e2e";

    buildInputs = [ goCache ];

    tags = [ "e2e" ];

    ldflags = [
      "-s"
      "-X github.com/edgelesssys/contrast/internal/manifest.TrustedMeasurement=${launchDigest}"
      "-X github.com/edgelesssys/contrast/internal/kuberesource.runtimeHandler=${runtimeHandler}"
    ];

    subPackages = [
      "e2e/genpolicy"
      "e2e/getdents"
      "e2e/openssl"
      "e2e/servicemesh"
      "e2e/release"
      "e2e/policy"
    ];
  };

  launchDigest = builtins.readFile "${microsoft.runtime-class-files}/launch-digest.hex";

  runtimeHandler = lib.removeSuffix "\n" (
    builtins.readFile "${microsoft.runtime-class-files}/runtime-handler"
  );

  packageOutputs = [
    "coordinator"
    "initializer"
    "cli"
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
        (path.append root "internal/attestation/snp/Milan.pem")
        (path.append root "internal/attestation/snp/Genoa.pem")
        (path.append root "node-installer")
        (fileset.difference (fileset.fileFilter (file: hasSuffix ".go" file.name) root) (
          path.append root "service-mesh"
        ))
      ];
    };

  proxyVendor = true;
  vendorHash = "sha256-6U7hL7xr0N+nkqfdlry352Pce5pX/SDUB4Iw/tc97sQ=";

  nativeBuildInputs = [ installShellFiles ];

  buildInputs = [ goCache ];

  subPackages = packageOutputs ++ [ "internal/kuberesource/resourcegen" ];

  prePatch = ''
    install -D ${lib.getExe genpolicy} cli/cmd/assets/genpolicy
    install -D ${genpolicy.settings-dev}/genpolicy-settings.json cli/cmd/assets/genpolicy-settings.json
    install -D ${genpolicy.rules}/genpolicy-rules.rego cli/cmd/assets/genpolicy-rules.rego
  '';

  CGO_ENABLED = 0;
  ldflags = [
    "-s"
    "-w"
    "-X main.version=v${version}"
    "-X main.genpolicyVersion=${genpolicy.version}"
    "-X github.com/edgelesssys/contrast/internal/manifest.TrustedMeasurement=${launchDigest}"
    "-X github.com/edgelesssys/contrast/internal/kuberesource.runtimeHandler=${runtimeHandler}"
  ];

  preCheck = ''
    export CGO_ENABLED=1
  '';

  checkPhase = ''
    runHook preCheck
    go test -race ./...
    runHook postCheck
  '';

  # doCheck = false;

  postInstall = ''
    for sub in ${builtins.concatStringsSep " " packageOutputs}; do
      mkdir -p "''${!sub}/bin"
      mv "$out/bin/$sub" "''${!sub}/bin/$sub"
    done

    # rename the cli binary to contrast
    mv "$cli/bin/cli" "$cli/bin/contrast"

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
    inherit e2e goCache;
    inherit (genpolicy) settings rules;
  };

  meta.mainProgram = "contrast";
}
