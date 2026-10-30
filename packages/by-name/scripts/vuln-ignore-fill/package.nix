# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  writeShellApplication,
  jq,
  coreutils,
  gawk,
}:

writeShellApplication {
  name = "vuln-ignore-fill";

  runtimeInputs = [
    jq
    coreutils
    gawk
  ];

  runtimeEnv = {
    ENTRIES_JQ = "${./entries.jq}";
    POSITION_AWK = "${./position.awk}";
  };

  text = builtins.readFile ./vuln-ignore-fill.sh;
}
