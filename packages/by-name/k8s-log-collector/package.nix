# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  writeShellApplication,
  symlinkJoin,
  inotify-tools,
  coreutils,
  findutils,
  gnused,
  gnugrep,
  systemdMinimal,
}:

let
  collect-pod-logs = writeShellApplication {
    name = "collect-pod-logs";
    runtimeInputs = [
      inotify-tools
      coreutils
      findutils
      gnugrep
      gnused
    ];
    text = builtins.readFile ./collect-pod-logs.sh;
  };

  # systemdMinimal disables all compression by default, but we need it
  # to read host journal files that may be compressed with LZ4/ZSTD.
  systemdWithJournal = systemdMinimal.override { withCompression = true; };

  collect-host-logs = writeShellApplication {
    name = "collect-host-logs";
    runtimeInputs = [
      coreutils
      systemdWithJournal
    ];
    text = builtins.readFile ./collect-host-logs.sh;
  };
in

symlinkJoin {
  name = "k8s-log-collector";
  paths = [
    collect-pod-logs
    collect-host-logs
  ];
}
