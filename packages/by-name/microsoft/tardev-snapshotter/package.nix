# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  rustPlatform,
  microsoft,
  cmake,
  protobuf,
}:

rustPlatform.buildRustPackage rec {
  pname = "tardev-snapshotter";
  inherit (microsoft.kata-runtime) version src;

  sourceRoot = "${src.name}/src/tardev-snapshotter";

  cargoHash = "sha256-TZWkv2k+W1+zl3YUxjfITAWvTN0M0JROEQJul2JCjhc=";

  nativeBuildInputs = [
    cmake
    protobuf
  ];

  env.RUSTC_BOOTSTRAP = 1;

  meta = {
    license = lib.licenses.asl20;
    mainProgram = "tardev-snapshotter";
  };
}
