# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  rustPlatform,
  runtime,
  protobuf,
  pkg-config,
  openssl,

  withDragonball ? false,
}:

rustPlatform.buildRustPackage (finalAttrs: {
  pname = "kata-runtime-rs";
  inherit (runtime) version src;

  buildAndTestSubdir = "src/runtime-rs";

  cargoLock = {
    lockFile = "${finalAttrs.src}/Cargo.lock";
    outputHashes = {
      "api_client-0.1.0" = "sha256-aWtVgYlcbssL7lQfMFGJah8DrJN0s/w1ZFncCPHT1aE=";
    };
  };

  postPatch = ''
    substitute src/runtime-rs/crates/shim/src/config.rs.in src/runtime-rs/crates/shim/src/config.rs \
      --replace-fail @PROJECT_NAME@ "Kata Containers" \
      --replace-fail @RUNTIME_VERSION@ ${finalAttrs.version} \
      --replace-fail @COMMIT@ none \
      --replace-fail @RUNTIME_NAME@ containerd-shim-kata-v2 \
      --replace-fail @CONTAINERD_RUNTIME_NAME@ io.containerd.kata.v2
  '';

  nativeBuildInputs = [
    pkg-config
    protobuf
  ];

  buildInputs = [
    openssl
    openssl.dev
  ];

  # Build.rs writes to src
  postConfigure = ''
    chmod -R +w .
  '';

  buildFeatures = lib.optional withDragonball "dragonball";

  env.OPENSSL_NO_VENDOR = 1;

  cargoTestFlags = [ "--bins" ];

  checkFlags = [
    # Tests need root privileges or other stuff not available in the sandbox.
    "--skip=device::device_manager::tests::test_new_block_device"
    "--skip=network::endpoint::endpoints_test::tests::test_ipvlan_construction"
    "--skip=network::endpoint::endpoints_test::tests::test_macvlan_construction"
    "--skip=network::endpoint::endpoints_test::tests::test_vlan_construction"
    "--skip=test::test_new_hypervisor"
  ];

  # This is a placeholder to make this package compatible with the Go runtime,
  # as the node-installer is configured to install this file.
  # TODO(katexochen): Remove when switching to runtime-rs.
  postInstall = ''
    echo "placeholder, kata-runtime doesn't exist for runtime-rs" > $out/bin/kata-runtime
  '';

  passthru = {
    inherit (runtime) cmdline;
  };

  meta = {
    changelog = "https://github.com/kata-containers/kata-containers/releases/tag/${finalAttrs.version}";
    homepage = "https://github.com/kata-containers/kata-containers";
    mainProgram = "containerd-shim-kata-v2";
    license = lib.licenses.asl20;
  };
})
