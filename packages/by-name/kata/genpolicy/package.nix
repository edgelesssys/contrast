# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  craneLib,
  source,
  openssl,
  pkg-config,
  protobuf,
  zlib,
  cmake,
  stdenv,
  stdenvNoCC,
  applyPatches,
}:

craneLib.buildPackage rec {
  pname = "genpolicy";
  inherit (source) version cargoVendorDir src;
  strictDeps = true;

  cargoExtraArgs = lib.concatStringsSep " " [
    "--target"
    stdenv.hostPlatform.rust.rustcTarget
    "--offline"
    "--package"
    "genpolicy"
  ];

  nativeBuildInputs = [
    cmake
    pkg-config
    protobuf
  ];

  buildInputs = [
    openssl
    openssl.dev
    zlib
  ];

  env = {
    OPENSSL_NO_VENDOR = 1;
    OPENSSL_DIR = "${openssl.dev}";
    OPENSSL_LIB_DIR = "${lib.getLib openssl}/lib";
  }
  // lib.optionalAttrs stdenv.hostPlatform.isStatic {
    "CARGO_TARGET_${stdenv.hostPlatform.rust.cargoEnvVarTarget}_RUSTFLAGS" =
      "-C target-feature=+crt-static -C link-arg=-static";
  };

  preBuild = ''
    chmod -R +w .
  ''
  + lib.optionalString stdenv.hostPlatform.isStatic ''
    unset NIX_CFLAGS_LINK
  '';

  cargoArtifacts = craneLib.buildDepsOnly {
    inherit
      pname
      version
      cargoVendorDir
      strictDeps
      cargoExtraArgs
      nativeBuildInputs
      buildInputs
      env
      preBuild
      ;
    src = source.srcRaw;
  };

  postPatch = ''
    make -C src/tools/genpolicy src/version.rs
  '';

  # TODO(sespiros): drop once kata-agent-policy compiles on Darwin upstream.
  doCheck = stdenv.hostPlatform.isLinux;

  # Only run library tests, the integration tests need internet access.
  cargoTestExtraArgs = "--package genpolicy --lib";

  passthru = rec {
    settings-base = stdenvNoCC.mkDerivation {
      name = "genpolicy-${version}-settings";
      inherit src;
      sourceRoot = "${src.name}";

      phases = [
        "unpackPhase"
        "patchPhase"
        "installPhase"
      ];
      installPhase = ''
        runHook preInstall
        install -D src/tools/genpolicy/genpolicy-settings.json $out/genpolicy-settings.json
        runHook postInstall
      '';
    };

    # We maintain two different patches for the genpolicy settings, one for development and one for
    # the release. We can't apply both to the Kata sources at the same time, so we have two
    # derivations here that apply the patches only to the settings file.
    #
    # If you need to modify these patches, this workflow may come in handy to keep diffs small.
    # Replace $CONTRAST with your repository worktree and adjust the patch file to _prod, if
    # needed.
    #
    #   cd $CONTRAST
    #   mkdir -p /tmp/a /tmp/b
    #   nix build .#base.kata.genpolicy.settings-base
    #   cp --no-preserve=mode result/genpolicy-settings.json /tmp/b
    #   cd /tmp/b
    #   patch -b -B ../a/ -p1 genpolicy-settings.json <$CONTRAST/packages/by-name/kata/genpolicy/genpolicy_settings_dev.patch
    #   # Now, edit /tmp/b/genpolicy-settings.json according to your needs.
    #   cd ..
    #   git diff --no-ext-diff --full-index --no-prefix a/genpolicy-settings.json b/genpolicy-settings.json >$CONTRAST/packages/by-name/kata/genpolicy/genpolicy_settings_dev.patch

    # These get applied on top of all the patches under the "runtime" folder
    settings = applyPatches {
      src = settings-base;
      patches = [ ./genpolicy_settings_prod.patch ];
    };

    # Settings that allow exec into CVM pods - not safe for production use!
    settings-dev = applyPatches {
      src = settings-base;
      patches = [ ./genpolicy_settings_dev.patch ];
    };

    # Switch to rules-allow-all to disable policy checks for debugging.
    rules = rules-prod;

    rules-prod = stdenvNoCC.mkDerivation {
      name = "genpolicy-${version}-rules";
      inherit src;
      sourceRoot = "${src.name}";

      phases = [
        "unpackPhase"
        "patchPhase"
        "installPhase"
      ];
      installPhase = ''
        runHook preInstall
        install -D src/tools/genpolicy/rules.rego $out/genpolicy-rules.rego
        runHook postInstall
      '';
    };

    rules-allow-all = stdenvNoCC.mkDerivation {
      name = "genpolicy-${version}-rules-allow-all";
      inherit src;
      sourceRoot = "${src.name}";

      phases = [
        "unpackPhase"
        "patchPhase"
        "installPhase"
      ];
      installPhase = ''
        runHook preInstall
        install -D ../../kata-opa/allow-all.rego $out/genpolicy-rules.rego
        runHook postInstall
      '';
    };
  };

  meta = {
    changelog = "https://github.com/kata-containers/kata-containers/releases/tag/${version}";
    homepage = "https://github.com/kata-containers/kata-containers";
    mainProgram = "genpolicy";
    license = lib.licenses.asl20;
  };
}
