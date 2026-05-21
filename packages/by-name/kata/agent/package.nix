# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  craneLib,
  rustPlatform,
  stdenv,
  source,
  cmake,
  pkg-config,
  protobuf,
  withSeccomp ? true,
  libseccomp,
  lvm2,
  openssl,
  withAgentPolicy ? true,
  withStandardOCIRuntime ? false,
  withInitData ? true,
}:

craneLib.buildPackage rec {
  pname = "kata-agent";
  inherit (source) version cargoVendorDir src;
  strictDeps = true;

  cargoExtraArgs = lib.concatStringsSep " " (
    [
      "--target"
      stdenv.hostPlatform.rust.rustcTarget
      "--offline"
      "--package"
      "kata-agent"
    ]
    ++ lib.optionals withSeccomp [
      "--features"
      "seccomp"
    ]
    ++ lib.optionals withAgentPolicy [
      "--features"
      "agent-policy"
    ]
    ++ lib.optionals withStandardOCIRuntime [
      "--features"
      "standard-oci-runtime"
    ]
    ++ lib.optionals withInitData [
      "--features"
      "init-data"
    ]
  );

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

  env = {
    LIBC = "gnu";
    OPENSSL_NO_VENDOR = 1;
  };

  cargoArtifacts = source.mkCargoArtifacts {
    inherit
      pname
      cargoExtraArgs
      strictDeps
      nativeBuildInputs
      buildInputs
      env
      ;
    stubPrefix = "src/agent/src";
    stubScript = ''
      printf 'fn main() {}\n' > $out/src/agent/src/main.rs
    '';
  };

  preBuild = ''
    chmod -R +w .
    ${source.restoreProtocolsSrc}
  '';

  postPatch = ''
    substitute src/agent/src/version.rs.in src/agent/src/version.rs \
      --replace-fail @AGENT_VERSION@ ${version} \
      --replace-fail @API_VERSION@ 0.0.1 \
      --replace-fail @VERSION_COMMIT@ ${version} \
      --replace-fail @COMMIT@ ""
  '';

  cargoTestExtraArgs = lib.concatStringsSep " " [
    "--"
    "--skip=mount::tests::test_already_baremounted"
    "--skip=mount::tests::test_mount"
    "--skip=netlink::tests::list_routes"
    "--skip=config::tests::test_from_cmdline_with_args_overwrites"
  ];

  # https://crates.io/crates/sev produces libsev.so, which is not needed for
  # the agent binary and pulls in a large dependency on rustc. Thus, we remove
  # it from the output.
  postInstall = ''
    rm -rf $out/lib
  '';

  meta = {
    description = ''The Kata agent is a long running process that runs inside the Virtual Machine (VM) (also known as the "pod" or "sandbox").'';
    license = lib.licenses.asl20;
    mainProgram = "kata-agent";
  };
}
