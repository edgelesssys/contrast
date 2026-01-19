# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  writeShellApplication,
  kubectl,
}:

# Usage: get-logs [start | download] $namespaceFile
writeShellApplication {
  name = "get-logs";

  runtimeInputs = [
    kubectl
  ];
  text = builtins.readFile ./get-logs.sh;
}
