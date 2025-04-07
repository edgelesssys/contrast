# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  kata,
  rustPlatform,
  openssl,
  pkg-config,
  protobuf,
  libiconv,
  zlib,
  cmake,
  stdenvNoCC,
  applyPatches,
}:

rustPlatform.buildRustPackage rec {
  pname = "genpolicy";
  inherit (kata.kata-runtime) version src;

  sourceRoot = "${src.name}/src/tools/genpolicy";

  cargoLock = {
    lockFile = "${src}/src/tools/genpolicy/Cargo.lock";
    outputHashes = {
      "tarfs-defs-0.1.0" = "sha256-J79fMuKOIVHEk6WvkLeM9IY5XQHyUJQOrwwMLvRvE60=";
    };
  };

  env.OPENSSL_NO_VENDOR = 1;

  nativeBuildInputs = [
    cmake
    pkg-config
    protobuf
  ];

  buildInputs = [
    openssl
    openssl.dev
    libiconv
    zlib
  ];

  # Build.rs writes to src
  postConfigure = ''
    chmod -R +w ../..
  '';

  preBuild = ''
    make src/version.rs
  '';

  # Only run library tests, the integration tests need internet access.
  cargoTestFlags = [ "--lib" ];

  passthru = rec {
    settings-base = stdenvNoCC.mkDerivation {
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

    settings = applyPatches {
      src = settings-base;
      patches = [ ./genpolicy_settings_prod.patch ];
    };

    settings-coordinator = applyPatches {
      src = settings-base;
      patches = [ ./genpolicy_settings_coordinator.patch ];
    };

    # Settings that allow exec into CVM pods - not safe for production use!
    settings-dev = applyPatches {
      src = settings-base;
      patches = [ ./genpolicy_settings_dev.patch ];
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

    rules-coordinator = rules;
  };

  meta = {
    changelog = "https://github.com/kata-containers/kata-containers/releases/tag/${version}";
    homepage = "https://github.com/kata-containers/kata-containers";
    mainProgram = "genpolicy";
    license = lib.licenses.asl20;
  };
}
