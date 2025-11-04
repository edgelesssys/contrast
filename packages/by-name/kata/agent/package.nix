# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  rustPlatform,
  runtime,
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
  inherit (runtime) version src;

  sourceRoot = "${src.name}/src/agent";

  cargoLock = {
    lockFile = "${src}/src/agent/Cargo.lock";
    outputHashes = {
      "cgroups-rs-0.3.5" = "sha256-BKD1ZPK5LqB/n2xD/oODArVKjbH+MQOeYn/UYbBHzn0=";
      "s390_pv_core-0.11.0" = "sha256-P275gUoF4JtaKvKPvzhCsBuo882kKCYebtNpCDEmTP0=";
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

  # https://crates.io/crates/sev produces libsev.so, which is not needed for
  # the agent binary and pulls in a large dependency on rustc. Thus, we remove
  # it from the output.
  postInstall = ''
    rm -rf $out/lib
  '';

  buildFeatures =
    lib.optional withSeccomp "seccomp"
    ++ lib.optional withAgentPolicy "agent-policy"
    ++ lib.optional withStandardOCIRuntime "standard-oci-runtime";

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
