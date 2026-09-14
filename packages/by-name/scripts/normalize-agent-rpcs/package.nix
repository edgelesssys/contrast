# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  writeShellApplication,
  jq,
}:

writeShellApplication {
  name = "normalize-agent-rpcs";
  runtimeInputs = [
    jq
  ];
  text = ''
    jq -s -S -f ${./normalize-agent-rpcs.jq} "$1"
  '';
}
