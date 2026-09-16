# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  writeShellApplication,
  cri-tools,
  gnused,
  jq,
  openssh,
  socat,
}:

# debugshell-host establishes an SSH connection from the node to the debugshell
# server running in a local Kata VM.
# It needs access to the containerd socket at the default location.
#
# Usage: debugshell-host $NAMESPACE $POD [command...]
writeShellApplication {
  name = "debugshell-host";
  text = ''
    export JQ_FIND_SANDBOX_ID=${./find_sandbox_id.jq}
  ''
  + builtins.readFile ./debugshell-host.sh;
  runtimeInputs = [
    cri-tools
    gnused
    jq
    openssh
    socat
  ];
}
