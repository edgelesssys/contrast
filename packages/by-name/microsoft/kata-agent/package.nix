# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  rustPlatform,
  microsoft,
  cmake,
  pkg-config,
  protobuf,
  withSeccomp ? true,
  libseccomp,
  lvm2,
  openssl,
  withAgentPolicy ? true,
  withStandardOCIRuntime ? false,
}:

rustPlatform.buildRustPackage rec {
  pname = "kata-agent";
  inherit (microsoft.kata-runtime) version src;

  sourceRoot = "${src.name}/src/agent";

  cargoLock = {
    lockFile = "${src}/src/agent/Cargo.lock";
    outputHashes = {
      "sev-1.2.1" = "sha256-5UkHDDJMVUG18AN/c6BSMTkEgSG8MBB33DZE355gXdE=";
      "regorus-0.1.4" = "sha256-hKhuPEgtVOW1/83fVyQB61ZPRYzNqPdDhS0lNyJpekc=";
    };
  };

  nativeBuildInputs = [
    cmake
    pkg-config
    protobuf
  ];

  buildInputs =
    [
      openssl
      openssl.dev
      lvm2.dev
      rustPlatform.bindgenHook
    ]
    ++ lib.optionals withSeccomp [
      libseccomp.dev
      libseccomp.lib
      libseccomp
    ];

  # Build.rs writes to src
  postConfigure = ''
    chmod -R +w ../..
  '';

  env = {
    LIBC = "gnu";
    SECCOMP = if withSeccomp then "yes" else "no";
    AGENT_POLICY = if withAgentPolicy then "yes" else "no";
    STANDARD_OCI_RUNTIME = if withStandardOCIRuntime then "yes" else "no";
    OPENSSL_NO_VENDOR = 1;
    RUST_BACKTRACE = 1;
  };

  buildPhase = ''
    runHook preBuild

    make

    runHook postBuild
  '';

  checkFlags = [
    "--skip=mount::tests::test_already_baremounted"
    "--skip=netlink::tests::list_routes stdout"
  ];

  meta = {
    description = ''The Kata agent is a long running process that runs inside the Virtual Machine (VM) (also known as the "pod" or "sandbox").'';
    license = lib.licenses.asl20;
    mainProgram = "kata-agent";
  };
}
