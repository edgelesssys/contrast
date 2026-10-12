# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  writeShellApplication,
  coreutils,
  kubectl,
  openssl,
  yq-go,
}:

# Usage: get-logs [start | download] $namespaceFile
writeShellApplication {
  name = "configure-imagepuller-mirror";

  runtimeInputs = [
    coreutils
    kubectl
    openssl
    yq-go
  ];
  text = builtins.readFile ./configure-imagepuller-mirror.sh;
}
