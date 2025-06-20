# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

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
      "attester-0.1.0" = "sha256-Sn3PfItlLkEOqfRqhGfBHryHzOROy3BaZXV2Hs9dKAU=";
      "cdi-0.1.0" = "sha256-DbXa6h678WYdBdQrVpetkfY8QzamW9lZIjd0u1fQgd4=";
      "loopdev-0.5.0" = "sha256-PD+iuZWPAFd3VUCgNB0ZrH/aCM2VMqJEyAv5/j1kqlA=";
      "sigstore-0.11.0" = "sha256-nmL9UQfebhBhgIm3WFWGsolK0ngOOl7d8Vo4XOZ7F0s=";
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
      --replace-fail @AGENT_VERSION@ ${version} \
      --replace-fail @API_VERSION@ 0.0.1 \
      --replace-fail @VERSION_COMMIT@ ${version} \
      --replace-fail @COMMIT@ ""

    # Disable LTO (Link Time Optimization) to reduce build time. The agent
    # binary shouldn't be that performance critical.
    substituteInPlace Cargo.toml \
      --replace-fail 'lto = true' 'lto = false'
  '';

  # Build.rs writes to src
  postConfigure = ''
    chmod -R +w ../..
  '';

  buildFeatures =
    lib.optional withSeccomp "seccomp"
    ++ lib.optional withAgentPolicy "agent-policy"
    ++ lib.optional withStandardOCIRuntime "standard-oci-runtime";

  env = {
    LIBC = "gnu";
    OPENSSL_NO_VENDOR = 1;
  };

  doCheck = false;

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
