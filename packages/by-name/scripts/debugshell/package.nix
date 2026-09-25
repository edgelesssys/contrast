# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  writeShellApplication,
  runCommand,
  symlinkJoin,
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
    runtimeEnv = {
      JQ_FIND_SANDBOX_ID = "${./find_sandbox_id.jq}";
    };
    text = builtins.readFile ./debugshell-host.sh;
    runtimeInputs = [
      cri-tools
      gnused
      jq
      openssh
      socat
    ];
  };

  # debugshell-guest establishes an SSH connection from within the VM.
  debugshell-guest = writeShellApplication {
    name = "debugshell";
    runtimeInputs = [ openssh ];
    text = ''
      if [[ ! -f ./id_ed25519 ]]; then
          ssh-keygen -t ed25519 -f ./id_ed25519 -N ""
      fi
      ssh -p 2222 \
          -o StrictHostKeyChecking=no \
          -o UserKnownHostsFile=/dev/null \
          -i ./id_ed25519 \
          root@localhost \
          "$@"
    '';
  };

  # debugshell-rootfs provides the expected directory structure and content for ssh to work.
  debugshell-rootfs = runCommand "debugshell-rootfs" { } ''
    mkdir -p \
      $out/etc \
      $out/tmp

    echo "root:x:0:0::/tmp:/bin/sh" > $out/etc/passwd
    echo "root:x:0:root" > $out/etc/group
  '';
in
symlinkJoin {
  name = "debugshell-tools";
  paths = [
    debugshell-host
    debugshell-guest
    debugshell-rootfs
  ];
}
