# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  rustPlatform,
  fetchFromGitHub,
  pkg-config,
  openssl,
  zlib,
  stdenv,
  darwin,
  cmake,
}:

rustPlatform.buildRustPackage rec {
  pname = "nydus";
  version = "2.2.3";

  src = fetchFromGitHub {
    owner = "dragonflyoss";
    repo = "nydus";
    rev = "v${version}";
    hash = "sha256-2iJQpOhWp93YFMwfPcPO9G6ZpJg/eP19gx09YJZR37k=";
  };

  postPatch = ''
    # Fixing 'error: call to `.deref()` on a reference in this situation does nothing'
    substituteInPlace src/bin/nydus-image/merge.rs \
      --replace-fail "MetadataTreeBuilder::parse_node(&rs, inode.deref(), path.to_path_buf())" "MetadataTreeBuilder::parse_node(&rs, inode, path.to_path_buf())" \
      --replace-fail "use std::ops::Deref;" ""
  '';

  cargoHash = "sha256-ahBnZ7tvaVwFCBr6kpeS8dgoZsnOJUR/KErfC3LaV9o=";

  nativeBuildInputs = [
    pkg-config
    cmake
  ];

  buildInputs = [
    openssl
    zlib
  ] ++ lib.optionals stdenv.isDarwin [ darwin.apple_sdk.frameworks.Security ];

  env = {
    OPENSSL_NO_VENDOR = true;
  };

  buildFeatures = [ "virtiofs" ];

  preCheck = ''
    export TEST_WORKDIR_PREFIX=$(mktemp -d)
  '';

  cargoCheckFlags = [ "--workspace" ];

  checkFlags = [
    "--skip integration"
    "--nocapture"
  ];

  meta = with lib; {
    description = "Nydus - the Dragonfly image service, providing fast, secure and easy access to container images";
    homepage = "https://github.com/dragonflyoss/nydus";
    license = with licenses; [
      bsd3
      asl20
    ];
  };
}
