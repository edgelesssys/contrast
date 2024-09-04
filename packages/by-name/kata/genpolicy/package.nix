# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  fetchurl,
  kata,
  rustPlatform,
  openssl,
  pkg-config,
  protobuf,
  libiconv,
  zlib,
  cmake,
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

  checkFlags = [
    # these want internet access, disable them
    "--skip=test_copyfile"
    "--skip=test_create_sandbox"
  ];

  passthru = {
    settings = fetchurl {
      name = "${pname}-${version}-settings";
      url = "https://raw.githubusercontent.com/kata-containers/kata-containers/${version}/src/tools/genpolicy/genpolicy-settings.json";
      hash = "sha256-kalmW/gWMJIWUNk7IzA0l1saMFu8QYb1DXZ8cU/QSxs=";
      downloadToTemp = true;
      recursiveHash = true;
      postFetch = "install -D $downloadedFile $out/genpolicy-settings.json";
    };

    # TODO(freax13): use real rules.
    rules = fetchurl {
      name = "${pname}-${version}-rules";
      url = "https://raw.githubusercontent.com/kata-containers/kata-containers/${version}/src/kata-opa/allow-all.rego";
      hash = "sha256-ubjA2RqoNurJphlH4wUNvdOxxtkvLlsaYfWsGYb9NLA=";
      downloadToTemp = true;
      recursiveHash = true;
      postFetch = "install -D $downloadedFile $out/genpolicy-rules.rego";
    };
  };

  meta = {
    changelog = "https://github.com/kata-containers/kata-containers/releases/tag/${version}";
    homepage = "https://github.com/kata-containers/kata-containers";
    mainProgram = "genpolicy";
    license = lib.licenses.asl20;
  };
}
