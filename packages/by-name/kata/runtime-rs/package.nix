{
  lib,
  rustPlatform,
  runtime,
  protobuf,
  pkg-config,
  openssl,

  withDragonball ? false,
}:

rustPlatform.buildRustPackage rec {
  pname = "kata-runtime-rs";
  inherit (runtime) version src;

  sourceRoot = "${src.name}/src/runtime-rs";

  cargoLock = {
    lockFile = "${src}/src/runtime-rs/Cargo.lock";
    outputHashes = {
      "api_client-0.1.0" = "sha256-aWtVgYlcbssL7lQfMFGJah8DrJN0s/w1ZFncCPHT1aE=";
    };
  };

  postPatch = ''
    substitute crates/shim/src/config.rs.in crates/shim/src/config.rs \
      --replace-fail @PROJECT_NAME@ "Kata Containers" \
      --replace-fail @RUNTIME_VERSION@ ${version} \
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
    chmod -R +w ../..
  '';

  buildFeatures = lib.optional withDragonball "dragonball";

  env.OPENSSL_NO_VENDOR = 1;

  cargoTestFlags = [
    "--lib"
    "--bins"
  ];

  checkFlags = [
    "--skip=device::device_manager::tests::test_new_block_device"
    "--skip=network::endpoint::endpoints_test::tests::test_ipvlan_construction"
    "--skip=network::endpoint::endpoints_test::tests::test_macvlan_construction"
    "--skip=network::endpoint::endpoints_test::tests::test_vlan_construction"
    "--skip=test::test_new_hypervisor"
  ];

  meta = {
    changelog = "https://github.com/kata-containers/kata-containers/releases/tag/${version}";
    homepage = "https://github.com/kata-containers/kata-containers";
    mainProgram = "genpolicy";
    license = lib.licenses.asl20;
  };
}
