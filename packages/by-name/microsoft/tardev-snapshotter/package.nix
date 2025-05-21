# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

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
  cargoHash = "sha256-VPqB/kQ1rk/bCeEEBMqjoNvp2rsAXr5smlIxWKcSVGE=";

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
