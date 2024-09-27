# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  rustPlatform,
  kata,
  cmake,
  pkg-config,
  protobuf,
  withSeccomp ? true,
  libseccomp,
  lvm2,
  openssl,
  withAgentPolicy ? true,
  withStandardOCIRuntime ? false,
  withGuestPull ? true,
}:

rustPlatform.buildRustPackage rec {
  pname = "kata-agent";
  inherit (kata.kata-runtime) version src;

  sourceRoot = "${src.name}/src/agent";

  cargoLock = {
    lockFile = "${src}/src/agent/Cargo.lock";
    outputHashes = {
      "attester-0.1.0" = "sha256-hx5Z5HxsyAPCQLY62koNGFHpG5M5PfG9Kagfsey58oI=";
      "loopdev-0.5.0" = "sha256-PD+iuZWPAFd3VUCgNB0ZrH/aCM2VMqJEyAv5/j1kqlA=";
      "sigstore-0.9.0" = "sha256-IeHuB5d5IU9YryeD47Qht0x806kJCoIOHsoEATRV+MY=";
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

  postPatch = ''
    substitute src/version.rs.in src/version.rs \
      --replace @@AGENT_VERSION@@ ${version} \
      --replace @@API_VERSION@@ 0.0.1 \
      --replace @@VERSION_COMMIT@@ ${version} \
      --replace @@COMMIT@@ "" \
      --replace @@AGENT_NAME@@ kata-agent \
      --replace @@AGENT_DIR@@ /usr/bin \
      --replace @@AGENT_PATH@@ /usr/bin/kata-agent
  '';

  # Build.rs writes to src
  postConfigure = ''
    chmod -R +w ../..
  '';

  buildFeatures =
    lib.optional withSeccomp "seccomp"
    ++ lib.optional withAgentPolicy "agent-policy"
    ++ lib.optional withStandardOCIRuntime "standard-oci-runtime"
    ++ lib.optional withGuestPull "guest-pull"
    ++ lib.optional (!withGuestPull) "default-pull";

  env = {
    LIBC = "gnu";
    OPENSSL_NO_VENDOR = 1;
  };

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
