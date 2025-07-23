# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  # keep-sorted start
  clang,
  cmake,
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
  pinnedCommit = "26c7f941aa3b378ded647709418f6ff4637bfb1c";
  toolchainFile = builtins.fetchurl {
    url = "https://raw.githubusercontent.com/kata-containers/kata-containers/${pinnedCommit}/rust-toolchain.toml";
    sha256 = "sha256:079kqn2kpsp7q5xjgxdhgxdqddvhacq75akwylfa1adjsjlynp1f";
  };

  toolchainSpec = {
    name = (lib.importTOML toolchainFile).toolchain.channel;
    sha256 = "sha256-Hn2uaQzRLidAWpfmRwSRdImifGUCAb9HeAqTYFXWeQk=";
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
