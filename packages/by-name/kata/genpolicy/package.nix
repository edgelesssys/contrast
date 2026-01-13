# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  runtime,
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
  inherit (runtime) version src;

  sourceRoot = "${src.name}/src/tools/genpolicy";

  cargoLock = {
    lockFile = "${src}/src/tools/genpolicy/Cargo.lock";
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
    #   nix build .#kata.genpolicy.settings-base
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

    rules-allow-all = stdenvNoCC.mkDerivation {
      name = "${pname}-${version}-rules-allow-all";
      inherit src sourceRoot;

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
