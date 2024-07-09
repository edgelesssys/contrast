# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  fetchFromGitHub,
  applyPatches,
  stdenvNoCC,
  rustPlatform,
  openssl,
  pkg-config,
  libiconv,
  zlib,
  cmake,
  protobuf,
}:

rustPlatform.buildRustPackage rec {
  pname = "genpolicy";
  version = "3.2.0.azl1.genpolicy0";

  src = applyPatches {
    src = fetchFromGitHub {
      owner = "microsoft";
      repo = "kata-containers";
      rev = "refs/tags/${version}";
      hash = "sha256-sFh2V7ylRDL6H50BcaHcgJAhrx4yvXzHNxtdQ9VYXdk=";
    };

    patches = [
      # Backport of https://github.com/kata-containers/kata-containers/pull/9706,
      # remove when picked up by Microsoft/kata-containers fork.
      ./0001-genpolicy-add-rules-and-types-for-volumeDevices.patch
      # Backport of https://github.com/kata-containers/kata-containers/pull/9725,
      # remove when picked up by Microsoft/kata-containers fork.
      ./0002-genpolicy-add-ability-to-filter-for-runtimeClassName.patch
      # Backport of https://github.com/kata-containers/kata-containers/pull/9864
      # remove when picked up by Microsoft/kata-containers fork.
      ./0003-genpolicy-allow-specifying-layer-cache-file.patch
    ];
  };

  sourceRoot = "${src.name}/src/tools/genpolicy";

  cargoHash = "sha256-YxIwsjs4K0TNVlwwA+PrOrCf16h7ZW+zU/jXeFfIMZo=";

  OPENSSL_NO_VENDOR = 1;

  nativeBuildInputs = [
    pkg-config
    cmake
    protobuf
  ];

  buildInputs = [
    openssl
    openssl.dev
    libiconv
    zlib
  ];

  preBuild = ''
    make src/version.rs
  '';

  passthru = rec {
    settings = stdenvNoCC.mkDerivation {
      name = "${pname}-${version}-settings";
      inherit src sourceRoot;

      phases = [
        "unpackPhase"
        "patchPhase"
        "installPhase"
      ];
      installPhase = ''
        runHook preInstall
        install -D genpolicy-settings.json $out/genpolicy-settings.json
        runHook postInstall
      '';
    };

    settings-coordinator = applyPatches {
      src = settings;
      patches = [ ./genpolicy_msft_settings_coordinator.patch ];
    };

    # Settings that allow exec into CVM pods - not safe for production use!
    settings-dev = applyPatches {
      src = settings;
      patches = [ ./genpolicy_msft_settings_dev.patch ];
    };

    rules = stdenvNoCC.mkDerivation {
      name = "${pname}-${version}-rules";
      inherit src sourceRoot;

      phases = [
        "unpackPhase"
        "patchPhase"
        "installPhase"
      ];
      installPhase = ''
        runHook preInstall
        install -D rules.rego $out/genpolicy-rules.rego
        runHook postInstall
      '';
    };

    rules-coordinator = applyPatches {
      src = rules;
      patches = [ ./genpolicy_msft_rules_coordinator.patch ];
    };
  };

  meta = {
    changelog = "https://github.com/microsoft/kata-containers/releases/tag/genpolicy-${version}";
    homepage = "https://github.com/microsoft/kata-containers";
    mainProgram = "genpolicy";
    licesnse = lib.licenses.asl20;
  };
}
