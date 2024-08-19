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
  fetchpatch,
}:

rustPlatform.buildRustPackage rec {
  pname = "kata-agent";
  inherit (kata.kata-runtime) version src;

  sourceRoot = "${src.name}/src/agent";

  cargoLock = {
    lockFile = "${src}/src/agent/Cargo.lock";
    outputHashes = {
      "attester-0.1.0" = "sha256-sRkBoBtE1irZxo5y3Ined6wMUmwxXq9c+Trt99q7kRk=";
      "loopdev-0.5.0" = "sha256-PD+iuZWPAFd3VUCgNB0ZrH/aCM2VMqJEyAv5/j1kqlA=";
      "sigstore-0.9.0" = "sha256-IeHuB5d5IU9YryeD47Qht0x806kJCoIOHsoEATRV+MY=";
    };
  };

  patches = [
    # Mount configfs into the workload container from the UVM.
    (fetchpatch {
      url = "https://github.com/kata-containers/kata-containers/commit/779152b91b20b22009d215887d06908c638d2efc.patch";
      stripLen = 2;
      hash = "sha256-gs1EgD+1Ol9rg0oo14WFQ3H7GCAU5EQrXSuQW+DtEWk=";
    })
  ];

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
