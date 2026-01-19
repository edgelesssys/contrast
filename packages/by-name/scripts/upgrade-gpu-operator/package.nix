# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  writeShellApplication,
  busybox,
  kubectl,
  kubernetes-helm,
}:

# Usage: upgrade-gpu-operator --version <version> [--blackwell]
writeShellApplication {
  name = "upgrade-gpu-operator";
  runtimeInputs = [
    busybox
    kubectl
    kubernetes-helm
  ];
  text = builtins.readFile ./upgrade-gpu-operator.sh;
}
