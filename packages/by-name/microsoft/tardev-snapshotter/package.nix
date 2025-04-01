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

  useFetchCargoVendor = true;
  cargoHash = "sha256-y1oxbE3OlpT8p3lJBV3+1CfrLHwwbHNbDOW2O6U3dA4=";

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
