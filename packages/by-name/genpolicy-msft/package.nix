# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ lib
, fetchFromGitHub
, fetchurl
, applyPatches
, rustPlatform
, openssl
, pkg-config
, libiconv
, zlib
, cmake
, protobuf
}:

rustPlatform.buildRustPackage rec {
  pname = "genpolicy";
  version = "3.2.0.azl1.genpolicy0";

  src = fetchFromGitHub {
    owner = "microsoft";
    repo = "kata-containers";
    rev = "refs/tags/${version}";
    hash = "sha256-W36RJFf0MVRIBV4ahpv6pqdAwgRYrlqmu4Y/8qiILS8=";
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
    settings = fetchurl {
      name = "${pname}-${version}-settings";
      # TODO(burgerdev): see whether future releases contain this file as an asset again (not true for 3.2.0.azl1).
      url = "https://raw.githubusercontent.com/microsoft/kata-containers/${version}/src/tools/genpolicy/genpolicy-settings.json";
      hash = "sha256-jrhzDqesm16yCV3aex48c2OcEimCUrxwhoaJUtAMPvo=";
      downloadToTemp = true;
      recursiveHash = true;
      postFetch = "install -D $downloadedFile $out/genpolicy-settings.json";
    };

    # Settings that allow exec into CVM pods - not safe for production use!
    settings-dev = applyPatches {
      src = settings;
      patches = [ ./genpolicy_msft_settings_dev.patch ];
    };

    rules = fetchurl {
      name = "${pname}-${version}-rules";
      # TODO(burgerdev): see whether future releases contain this file as an asset again (not true for 3.2.0.azl1).
      url = "https://raw.githubusercontent.com/microsoft/kata-containers/${version}/src/tools/genpolicy/rules.rego";
      hash = "sha256-fhE5hDND5QeZtEw3u+qgSVsFO+00cc41k/r/Y+km6TU=";
      downloadToTemp = true;
      recursiveHash = true;
      postFetch = "install -D $downloadedFile $out/genpolicy-rules.rego";
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
