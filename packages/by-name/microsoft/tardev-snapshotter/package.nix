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

  cargoHash = "sha256-d3bDtkB+9pL2Zq5ZGG3Q3bG0YAWBVpYw1ZthacGeUyY=";

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
