# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  rustPlatform,
  kata,
  applyPatches,
  cmake,
  craneLib,
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
let
  inherit (kata.kata-runtime) version;
  cleanSource =
    src:
    (applyPatches {
      src =
        let
          additionalFilter = path: _type: builtins.match ".*[in|proto]$" path != null;
          additionalOrCargo =
            path: type: (additionalFilter path type) || (craneLib.filterCargoSources path type);
        in
        lib.cleanSourceWith {
          inherit src;
          filter = additionalOrCargo;
          name = "source";
        };

      postPatch = ''
        substitute src/agent/src/version.rs.in src/agent/src/version.rs \
         --replace-fail @AGENT_VERSION@ ${version} \
         --replace-fail @API_VERSION@ 0.0.1 \
         --replace-fail @VERSION_COMMIT@ ${version} \
         --replace-fail @COMMIT@ ""

        substituteInPlace src/agent/Cargo.toml \
         --replace-fail 'lto = true' 'lto = false'
      '';
    });
in
craneLib.buildPackage rec {
  inherit version;
  pname = "kata-agent";
  strictDeps = true;

  outputHashes = {
    "git+https://github.com/cncf-tags/container-device-interface-rs?rev=fba5677a8e7cc962fc6e495fcec98d7d765e332a#fba5677a8e7cc962fc6e495fcec98d7d765e332a" =
      "sha256-DbXa6h678WYdBdQrVpetkfY8QzamW9lZIjd0u1fQgd4=";
    "git+https://github.com/confidential-containers/guest-components?rev=0a06ef241190780840fbb0542e51b198f1f72b0b#0a06ef241190780840fbb0542e51b198f1f72b0b" =
      "sha256-Bp8Ny9wqS2iDqZCiW2DUkgTGq3h1DJ92CZT9LCZx/h0=";
    "git+https://github.com/ibm-s390-linux/s390-tools?rev=4942504a9a2977d49989a5e5b7c1c8e07dc0fa41#4942504a9a2977d49989a5e5b7c1c8e07dc0fa41" =
      "sha256-P275gUoF4JtaKvKPvzhCsBuo882kKCYebtNpCDEmTP0=";
    "git+https://github.com/mdaffin/loopdev?rev=c9f91e8f0326ce8a3364ac911e81eb32328a5f27#c9f91e8f0326ce8a3364ac911e81eb32328a5f27" =
      "sha256-PD+iuZWPAFd3VUCgNB0ZrH/aCM2VMqJEyAv5/j1kqlA=";
    "git+https://github.com/sigstore/sigstore-rs.git?rev=c39c519#c39c519dd99be23f18e6143dd233b46bf2096e4d" =
      "sha256-nmL9UQfebhBhgIm3WFWGsolK0ngOOl7d8Vo4XOZ7F0s=";
  };

  postUnpack = ''
    cd $sourceRoot/src/agent
    sourceRoot="."
  '';

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

  doCheck = false;

  cargoArtifacts = craneLib.buildDepsOnly rec {
    inherit
      version
      strictDeps
      outputHashes
      postUnpack
      nativeBuildInputs
      buildInputs
      doCheck
      ;
    src = cleanSource kata.kata-runtime.src.src;
    cargoToml = "${src}/src/agent/Cargo.toml";
    cargoLock = "${src}/src/agent/Cargo.lock";
  };

  src = cleanSource kata.kata-runtime.src;
  cargoToml = "${cleanSource kata.kata-runtime.src.src}/src/agent/Cargo.toml";
  cargoLock = "${cleanSource kata.kata-runtime.src.src}/src/agent/Cargo.lock";

  # https://crates.io/crates/sev produces libsev.so, which is not needed for
  # the agent binary and pulls in a large dependency on rustc. Thus, we remove
  # it from the output.
  postInstall = ''
    rm -rf $out/lib
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

  test = craneLib.cargoTest {
    inherit
      cargoArtifacts
      version
      strictDeps
      outputHashes
      postUnpack
      nativeBuildInputs
      buildInputs
      src
      cargoToml
      cargoLock
      ;
    doCheck = true;
    cargoTestExtraArgs = "-- --skip=mount::tests::test_already_baremounted --skip=netlink::tests::list_routes --skip=initdata::tests::parse_initdata";
  };

  meta = {
    description = ''The Kata agent is a long running process that runs inside the Virtual Machine (VM) (also known as the "pod" or "sandbox").'';
    license = lib.licenses.asl20;
    mainProgram = "kata-agent";
  };
}
