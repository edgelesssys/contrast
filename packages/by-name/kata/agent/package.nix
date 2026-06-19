# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  source,
  buildCargoSbom,
  withSeccomp ? true,
  withAgentPolicy ? true,
  withStandardOCIRuntime ? false,
  withInitData ? true,
}:

let
  features =
    lib.optional withSeccomp "seccomp"
    ++ lib.optional withAgentPolicy "agent-policy"
    ++ lib.optional withStandardOCIRuntime "standard-oci-runtime"
    ++ lib.optional withInitData "init-data";
in

(source.cargoNixPackage.workspaceMembers."kata-agent".build.override {
  inherit features;
  runTests = true;
  # The test framework's anyhow assertions stringify backtraces; suppress to
  # match crane's default of no backtrace.
  testPreRun = ''
    unset RUST_BACKTRACE
  '';
  testCrateFlags = [
    "--skip=mount::tests::test_already_baremounted"
    "--skip=mount::tests::test_mount"
    "--skip=netlink::tests::list_routes"
    "--skip=config::tests::test_from_cmdline_with_args_overwrites"
  ];
}).overrideAttrs
  (prev: {
    pname = "kata-agent";
    passthru = (prev.passthru or { }) // {
      bombonVendoredSbom = buildCargoSbom {
        inherit (source) cargoNixPackage;
        member = "kata-agent";
        inherit features;
      };
    };
    meta = (prev.meta or { }) // {
      description = ''The Kata agent is a long running process that runs inside the Virtual Machine (VM) (also known as the "pod" or "sandbox").'';
      license = lib.licenses.asl20;
      mainProgram = "kata-agent";
    };
  })
