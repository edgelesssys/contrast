# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  runCommand,
  symlinkJoin,
  writeShellApplication,
  cri-tools,
  gnused,
  jq,
  openssh,
  socat,
}:

let

  # debugshell-host establishes an SSH connection from the node to the debugshell
  # server running in a local Kata VM.
  # It needs access to the containerd socket at the default location.
  #
  # Usage: debugshell-host $NAMESPACE $POD [command...]
  debugshell-host = writeShellApplication {
    name = "debugshell-host";
    text = builtins.readFile ./debugshell-host.sh;
    runtimeInputs = [
      cri-tools
      gnused
      jq
      openssh
      socat
    ];
  };

  debugshell-config = runCommand "debugshell-rootfs" { } ''
    mkdir -p \
      $out/etc \
      $out/tmp

      echo "root:x:0:0::/tmp:/bin/sh" > $out/etc/passwd
      echo "root:x:0:root" > $out/etc/group
  '';

in
symlinkJoin {
  name = "debugshell-rootfs";
  paths = [
    debugshell-host
    debugshell-config
  ];
}
