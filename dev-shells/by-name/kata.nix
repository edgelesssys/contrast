# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  # keep-sorted start
  clang,
  cmake,
  contrastPkgs,
  fenix,
  lib,
  lld,
  llvm,
  llvmPackages,
  pkg-config,
  protobuf,
  # keep-sorted end
  mkShell,
  pkgsCross,
}:

let
  toolchainFile = "${contrastPkgs.kata.runtime.src}/rust-toolchain.toml";
  toolchainSpec = {
    name = (lib.importTOML toolchainFile).toolchain.channel;
    sha256 = "sha256-SBKjxhC6zHTu0SyJwxLlQHItzMzYZ71VCWQC2hOzpRY=";
  };
  muslToolchain = fenix.targets."x86_64-unknown-linux-musl".fromToolchainName toolchainSpec;
in
mkShell {
  buildInputs = [
    # keep-sorted start
    clang
    cmake
    lld
    llvm
    llvmPackages.libclang
    pkg-config
    protobuf
    # keep-sorted end
    pkgsCross.musl64.buildPackages.gcc
    (fenix.combine [
      ((fenix.fromToolchainName toolchainSpec).withComponents [
        "cargo"
        "clippy"
        "rust-src"
        "rustc"
        "rustfmt"
        "rust-analyzer"
      ])
      muslToolchain.rust-std
    ])
  ];

  shellHook = ''
    export CC_x86_64_unknown_linux_musl=${pkgsCross.musl64.buildPackages.gcc}/bin/x86_64-unknown-linux-musl-gcc
    export AR_x86_64_unknown_linux_musl=${pkgsCross.musl64.buildPackages.gcc}/bin/x86_64-unknown-linux-musl-ar
    export LIBCLANG_PATH=${llvmPackages.libclang.lib}/lib
  '';
}
