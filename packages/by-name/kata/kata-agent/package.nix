# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{ lib
, rustPlatform
, fetchFromGitHub
, cmake
, pkg-config
, protobuf
, withSeccomp ? true
, libseccomp
, lvm2
, openssl
, withAgentPolicy ? true
, withStandardOCIRuntime ? false
}:

rustPlatform.buildRustPackage rec {
  pname = "kata-agent";
  version = "3.5.0";

  src = fetchFromGitHub {
    owner = "kata-containers";
    repo = "kata-containers";
    rev = version;
    hash = "sha256-5pIJpyeydOVA+GrbCvNqJsmK3zbtF/5iSJLI2C1wkLM=";
  };

  sourceRoot = "${src.name}/src/agent";

  cargoLock = {
    lockFile = "${src}/src/agent/Cargo.lock";
    outputHashes = {
      "image-rs-0.1.0" = "sha256-L+tGVqCv3i4c72GY0KhCYq5brgGjAUGKED+9+qjr714=";
      "loopdev-0.5.0" = "sha256-PD+iuZWPAFd3VUCgNB0ZrH/aCM2VMqJEyAv5/j1kqlA=";
      "sigstore-0.8.0" = "sha256-lmcokyIx4R84miC8Rf3NjV3QS6XffbhzQeZGCM0u7lc=";
    };
  };

  nativeBuildInputs = [
    cmake
    pkg-config
    protobuf
  ];

  buildInputs = [
    openssl
    openssl.dev
    lvm2.dev
    rustPlatform.bindgenHook
  ] ++ lib.optionals withSeccomp [
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
