# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  # keep-sorted start
  azure-cli,
  crane,
  delve,
  go,
  golangci-lint,
  gopls,
  gotools,
  just,
  kubectl,
  yq-go,
  # keep-sorted end
  mkShell,
}:

mkShell {
  packages = [
    # keep-sorted start
    azure-cli
    crane
    delve
    go
    golangci-lint
    gopls
    gotools
    just
    kubectl
    yq-go
    # keep-sorted end
  ];
  shellHook = ''
    alias make=just
    export DO_NOT_TRACK=1
  '';
}
