# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  buildGoModule,
  source,
  yq-go,
  git,
}:

buildGoModule (finalAttrs: {
  pname = "kata-runtime";

  inherit (source) version src;

  sourceRoot = "${finalAttrs.src.name}/src/runtime";

  vendorHash = "sha256-TVIWv+b94z3QuhAxRbRpCuMxQtE8Nq2fuQ4R9I6UqMs=";

  subPackages = [
    "cmd/containerd-shim-kata-v2"
    "cmd/kata-monitor"
    "cmd/kata-runtime"
  ];

  preBuild = ''
    substituteInPlace Makefile \
      --replace-fail 'include golang.mk' ""
    for f in $(find . -name '*.in' -type f); do
      make ''${f%.in}
    done
  '';

  nativeBuildInputs = [
    yq-go
    git
  ];

  ldflags = [ "-s" ];

  # Hack to skip all kata-runtime tests, which require a Git repo.
  preCheck = ''
    rm cmd/kata-runtime/*_test.go
  '';

  checkFlags =
    let
      # Skip tests that require a working hypervisor
      skippedTests = [
        "TestArchRequiredKernelModules"
        "TestCheckCLIFunctionFail"
        "TestEnvCLIFunction(|Fail)"
        "TestEnvGetAgentInfo"
        "TestEnvGetEnvInfo(|SetsCPUType|NoHypervisorVersion|AgentError|NoOSRelease|NoProcCPUInfo|NoProcVersion)"
        "TestEnvGetRuntimeInfo"
        "TestEnvHandleSettings(|InvalidParams)"
        "TestGetHypervisorInfo"
        "TestGetHypervisorInfoSocket"
        "TestSetCPUtype"
      ];
    in
    [ "-skip=^${builtins.concatStringsSep "$|^" skippedTests}$" ];

  # Default commandline used by Kata containers.
  # This has no single source upstream, but can be derived by manually checking what command-line
  # is used when Kata starts a VM.
  # For example, this command should do the job:
  # `journalctl -t kata -l --no-pager | grep launching | tail -1`
  #
  # Notice! Ordering matters and depends on the order in which kata-runtime adds these values.
  passthru.cmdline = {
    prefix =
      debug:
      [
        "tsc=reliable"
        "no_timer_check"
        "rcupdate.rcu_expedited=1"
        "i8042.direct=1"
        "i8042.dumbkbd=1"
        "i8042.nopnp=1"
        "i8042.noaux=1"
        "noreplace-smp"
        "reboot=k"
        "cryptomgr.notests"
        "net.ifnames=0"
        "pci=lastbus=0"
        "root=/dev/vda1"
        "rootflags=ro"
        "rootfstype=erofs"
      ]
      # In debug mode, use legacy serial (ttyS0) instead of virtio console (hvc0/hvc1)
      # to capture OVMF firmware output via kata's console watcher.
      # This must match use_legacy_serial=true set in kataconfig/config.go for debug.
      ++ lib.optionals (!debug) [
        "console=hvc0"
        "console=hvc1"
      ]
      ++ lib.optionals debug [
        "console=ttyS0"
      ]
      ++ lib.optionals debug [
        "debug"
        "systemd.show_status=true"
        "systemd.log_level=debug"
      ]
      ++ lib.optionals (!debug) [
        "quiet"
        "systemd.show_status=false"
      ]
      ++ [
        "panic=1"
        "selinux=0"
        "systemd.unit=kata-containers.target"
        "systemd.mask=systemd-networkd.service"
        "systemd.mask=systemd-networkd.socket"
        "scsi_mod.scan=none"
      ]
      ++ lib.optionals debug [
        "agent.log=debug"
        "agent.debug_console"
        "agent.debug_console_vport=1026"
      ]
      ++ [ "agent.launch_process_timeout=6" ];
    suffix = _debug: [ ];
  };

  # Output hashes for the git dependencies in the Kata workspace's Cargo.lock.
  # Shared by all Rust packages built from this source (kata-agent, genpolicy,
  # runtime-rs) and by kata.crate-sboms, so they only need to be maintained here.
  passthru.cargoOutputHashes = {
    "api_client-0.1.0" = "sha256-RdwQg6/EI+oGkyNXnu5t1q87oTXev25XpIaE+PWDTx4=";
    "cgroups-rs-0.3.5" = "sha256-BKD1ZPK5LqB/n2xD/oODArVKjbH+MQOeYn/UYbBHzn0=";
    "micro_http-0.1.0" = "sha256-XemdzwS25yKWEXJcRX2l6QzD7lrtroMeJNOUEWGR7WQ=";
    "regorus-0.9.1" = "sha256-+TCq9r8kTNM0URbcDP4D9/lKA6Bni7+KgrGRTJFbQPM=";
    "s390_pv_core-0.11.0" = "sha256-P275gUoF4JtaKvKPvzhCsBuo882kKCYebtNpCDEmTP0=";
  };

  meta.mainProgram = "containerd-shim-kata-v2";
})
