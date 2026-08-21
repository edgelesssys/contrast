# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  writeShellApplication,
  coreutils,
  gnugrep,
  jq,
  kubectl,
  kubernetes-helm,
}:

# Usage: upgrade-gpu-operator --version <version>
writeShellApplication {
  name = "upgrade-gpu-operator";
  runtimeInputs = [
    coreutils
    gnugrep
    jq
    kubectl
    kubernetes-helm
  ];
  text = builtins.readFile ./upgrade-gpu-operator.sh;
}
