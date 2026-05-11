# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  craneLib,
  source,
  stdenv,
  protobuf,
  pkg-config,
  openssl,

  withDragonball ? false,
}:

craneLib.buildPackage rec {
  pname = "kata-runtime-rs";
  inherit (source) version cargoVendorDir src;
  strictDeps = true;

  cargoExtraArgs = lib.concatStringsSep " " (
    [
      "--target"
      stdenv.hostPlatform.rust.rustcTarget
      "--offline"
      "--package"
      "shim"
    ]
    ++ lib.optionals withDragonball [
      "--features"
      "dragonball"
    ]
  );

  nativeBuildInputs = [
    pkg-config
    protobuf
  ];

  buildInputs = [
    openssl
    openssl.dev
  ];

  env = {
    OPENSSL_NO_VENDOR = 1;
  };

  preBuild = ''
    chmod -R +w .
  '';

  cargoArtifacts = craneLib.buildDepsOnly {
    inherit
      pname
      version
      cargoVendorDir
      strictDeps
      cargoExtraArgs
      nativeBuildInputs
      buildInputs
      env
      preBuild
      ;
    src = source.srcRaw;
  };

  postPatch = ''
    substitute src/runtime-rs/crates/shim/src/config.rs.in src/runtime-rs/crates/shim/src/config.rs \
      --replace-fail @PROJECT_NAME@ "Kata Containers" \
      --replace-fail @RUNTIME_VERSION@ ${version} \
      --replace-fail @COMMIT@ none \
      --replace-fail @RUNTIME_NAME@ containerd-shim-kata-v2 \
      --replace-fail @CONTAINERD_RUNTIME_NAME@ io.containerd.kata.v2
  '';

  cargoTestExtraArgs = lib.concatStringsSep " " [
    "--bins"
    "--"
    # Tests need root privileges or other stuff not available in the sandbox.
    "--skip=device::device_manager::tests::test_new_block_device"
    "--skip=network::endpoint::endpoints_test::tests::test_ipvlan_construction"
    "--skip=network::endpoint::endpoints_test::tests::test_macvlan_construction"
    "--skip=network::endpoint::endpoints_test::tests::test_vlan_construction"
    "--skip=test::test_new_hypervisor"
  ];

  # This is a placeholder to make this package compatible with the Go runtime,
  # as the node-installer is configured to install this file.
  # TODO(burgerdev): Remove when switching to runtime-rs.
  postInstall = ''
    echo "placeholder, kata-runtime doesn't exist for runtime-rs" > $out/bin/kata-runtime
  '';

  # TODO(burgerdev): test debug cmdline
  # TODO(burgerdev): this should be provided by Kata directly.
  passthru.cmdline = {
    prefix = _debug: [
      "reboot=k"
      "panic=1"
      "systemd.unit=kata-containers.target"
      "systemd.mask=systemd-networkd.service"
      "systemd.mask=systemd-networkd.socket"
    ];
    suffix = _debug: [
      "selinux=0"
      "console=hvc0"
    ];
  };

  meta = {
    changelog = "https://github.com/kata-containers/kata-containers/releases/tag/${version}";
    homepage = "https://github.com/kata-containers/kata-containers";
    mainProgram = "containerd-shim-kata-v2";
    license = lib.licenses.asl20;
  };
}
