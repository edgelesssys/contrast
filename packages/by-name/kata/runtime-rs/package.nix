# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  source,
  runCommand,
  withDragonball ? false,
}:

let
  shim = source.cargoNixPackage.workspaceMembers."shim".build.override {
    features = lib.optional withDragonball "dragonball";
    runTests = true;
    testCrateFlags = [
      "--skip=device::device_manager::tests::test_new_block_device"
      "--skip=network::endpoint::endpoints_test::tests::test_ipvlan_construction"
      "--skip=network::endpoint::endpoints_test::tests::test_macvlan_construction"
      "--skip=network::endpoint::endpoints_test::tests::test_vlan_construction"
      "--skip=test::test_new_hypervisor"
    ];
  };
in
runCommand "kata-runtime-rs-${source.version}"
  {
    passthru.version = source.version;
    passthru.src = source.src;
    passthru.cmdline = {
      prefix = _debug: [
        "reboot=k"
        "panic=1"
        "systemd.unit=kata-containers.target"
        "systemd.mask=systemd-networkd.service"
        "systemd.mask=systemd-networkd.socket"
        "agent.cdh_api_timeout=50"
        "agent.launch_process_timeout=6"
      ];
      suffix = _debug: [
        "selinux=0"
        "console=hvc0"
      ];
    };
    meta = {
      changelog = "https://github.com/kata-containers/kata-containers/releases/tag/${source.version}";
      homepage = "https://github.com/kata-containers/kata-containers";
      mainProgram = "containerd-shim-kata-v2";
      license = lib.licenses.asl20;
    };
  }
  ''
    mkdir -p $out/bin
    cp ${shim}/bin/* $out/bin/
    # Placeholder for Go runtime compatibility; the node-installer expects this file.
    # TODO(burgerdev): Remove when switching to runtime-rs.
    echo "placeholder, kata-runtime doesn't exist for runtime-rs" > $out/bin/kata-runtime
  ''
